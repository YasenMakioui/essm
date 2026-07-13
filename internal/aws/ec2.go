package aws

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type instance struct {
	id    string
	name  string
	state string
}

func (i instance) GetStringData() []string {
	// returns a slice with the attributes {id,}

	return []string{i.id, i.name, i.state}
}

// represents all instances

type instanceData struct {
	Instances []instance
}

// init struct, performs a req to aws api to gather that info

func NewInstanceData() instanceData {
	i := instanceData{}

	i.update()

	return i
}

func (i *instanceData) update() {
	var ec2Instances []instance

	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		log.Fatal(err)
	}

	client := ec2.NewFromConfig(cfg)
	instanceInput := &ec2.DescribeInstancesInput{}
	instanceOutput, err := client.DescribeInstances(context.TODO(), instanceInput)

	if err != nil {
		log.Fatal("Could not load credentials")
	}

	// iterate over the response to get the instances
	for _, object := range instanceOutput.Reservations {
		for _, ec2instance := range object.Instances {
			// get instance ID
			instanceId := aws.ToString(ec2instance.InstanceId)

			// iterate over tags to fins Name tag, by default use NoInstanceName
			instanceName := "NoInstanceName"

			for _, v := range ec2instance.Tags {
				if aws.ToString(v.Key) == "Name" {
					instanceName = aws.ToString(v.Value)
					break
				}
			}

			instanceState := ec2instance.State.Name

			ec2Instances = append(ec2Instances, instance{id: instanceId, name: instanceName, state: string(instanceState)})
		}
	}

	i.Instances = ec2Instances
}
