package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// SSMStatus reports whether an instance is reachable through SSM.
type SSMStatus int

const (
	SSMUnknown SSMStatus = iota
	SSMOnline
	SSMOffline
)

// Instance is the domain type. It carries no UI concerns (no table.Row).
type Instance struct {
	ID    string
	Name  string
	State string    // ec2 lifecycle: running/stopped/…
	SSM   SSMStatus // can we start-session to it?
}

// FetchInstances lists EC2 instances and marks which are reachable via SSM.
func FetchInstances(ctx context.Context) ([]Instance, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	instances, err := describeInstances(ctx, cfg)
	if err != nil {
		return nil, err
	}

	online, err := ssmOnlineSet(ctx, cfg)
	if err != nil {
		return nil, err
	}

	for i := range instances {
		if online[instances[i].ID] {
			instances[i].SSM = SSMOnline
		} else {
			instances[i].SSM = SSMOffline
		}
	}

	return instances, nil
}

func describeInstances(ctx context.Context, cfg aws.Config) ([]Instance, error) {
	client := ec2.NewFromConfig(cfg)

	var instances []Instance
	p := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})
	for p.HasMorePages() {
		out, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe instances: %w", err)
		}
		for _, res := range out.Reservations {
			for _, in := range res.Instances {
				state := ""
				if in.State != nil {
					state = string(in.State.Name)
				}
				instances = append(instances, Instance{
					ID:    aws.ToString(in.InstanceId),
					Name:  nameTag(in.Tags),
					State: state,
				})
			}
		}
	}
	return instances, nil
}

func nameTag(tags []ec2types.Tag) string {
	for _, t := range tags {
		if aws.ToString(t.Key) == "Name" {
			return aws.ToString(t.Value)
		}
	}
	return "NoInstanceName"
}
