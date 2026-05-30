package externalstorage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.temporal.io/sdk/contrib/aws/s3driver"
	"go.temporal.io/sdk/contrib/aws/s3driver/awssdkv2"
	"go.temporal.io/sdk/converter"
)

func newS3Driver(ctx context.Context, cfg S3Config) (converter.StorageDriver, error) {
	c, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, err
	}

	return s3driver.NewDriver((s3driver.Options{
		Client: awssdkv2.NewClient(s3.NewFromConfig(c)),
		Bucket: s3driver.StaticBucket(cfg.Bucket),
	}))
}
