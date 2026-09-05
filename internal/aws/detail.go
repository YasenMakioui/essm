package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Volume struct {
	Device string
	ID     string
	Status string
}

type SecurityGroup struct {
	ID   string
	Name string
}

type InstanceDetail struct {
	ID             string
	Type           string
	AMI            string
	PrivateIP      string
	PublicIP       string
	SubnetID       string
	VpcID          string
	IAMProfile     string // instance-profile name parsed from the ARN
	Volumes        []Volume
	SecurityGroups []SecurityGroup
	Tags           map[string]string
}

// FetchInstanceDetail loads full detail for one instance. Called lazily (on "i"),
// not during the list refresh, so the main table stays cheap.
func FetchInstanceDetail(ctx context.Context, id string) (InstanceDetail, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return InstanceDetail{}, fmt.Errorf("load aws config: %w", err)
	}

	client := ec2.NewFromConfig(cfg)
	out, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{id},
	})
	if err != nil {
		return InstanceDetail{}, fmt.Errorf("describe instance %s: %w", id, err)
	}

	for _, res := range out.Reservations {
		for _, in := range res.Instances {
			return toDetail(in), nil
		}
	}
	return InstanceDetail{}, fmt.Errorf("instance %s not found", id)
}

func toDetail(in ec2types.Instance) InstanceDetail {
	d := InstanceDetail{
		ID:        aws.ToString(in.InstanceId),
		Type:      string(in.InstanceType),
		AMI:       aws.ToString(in.ImageId),
		PrivateIP: aws.ToString(in.PrivateIpAddress),
		PublicIP:  aws.ToString(in.PublicIpAddress),
		SubnetID:  aws.ToString(in.SubnetId),
		VpcID:     aws.ToString(in.VpcId),
		Tags:      map[string]string{},
	}
	if in.IamInstanceProfile != nil {
		d.IAMProfile = profileName(aws.ToString(in.IamInstanceProfile.Arn))
	}
	for _, bdm := range in.BlockDeviceMappings {
		v := Volume{Device: aws.ToString(bdm.DeviceName)}
		if bdm.Ebs != nil {
			v.ID = aws.ToString(bdm.Ebs.VolumeId)
			v.Status = string(bdm.Ebs.Status)
		}
		d.Volumes = append(d.Volumes, v)
	}

	for _, sg := range in.SecurityGroups {
		d.SecurityGroups = append(d.SecurityGroups, SecurityGroup{
			ID:   aws.ToString(sg.GroupId),
			Name: aws.ToString(sg.GroupName),
		})
	}
	for _, t := range in.Tags {
		d.Tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	return d
}

// profileName turns arn:aws:iam::acct:instance-profile/Name into Name.
func profileName(arn string) string {
	if i := strings.LastIndex(arn, "/"); i >= 0 {
		return arn[i+1:]
	}
	return arn
}
