package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// ssmOnlineSet returns the set of instance IDs whose SSM agent is Online.
func ssmOnlineSet(ctx context.Context, cfg aws.Config) (map[string]bool, error) {
	client := ssm.NewFromConfig(cfg)

	online := make(map[string]bool)
	p := ssm.NewDescribeInstanceInformationPaginator(client, &ssm.DescribeInstanceInformationInput{})
	for p.HasMorePages() {
		out, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describe ssm instance information: %w", err)
		}
		for _, info := range out.InstanceInformationList {
			if info.PingStatus == ssmtypes.PingStatusOnline {
				online[aws.ToString(info.InstanceId)] = true
			}
		}
	}
	return online, nil
}
