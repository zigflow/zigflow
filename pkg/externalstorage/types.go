package externalstorage

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/converter"
)

type Config struct {
	Type                 StorageType
	PayloadSizeThreshold int

	S3 S3Config
}

type S3Config struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	UsePathStyle    bool
}

func New(ctx context.Context, cfg Config) (st converter.ExternalStorage, err error) {
	st.PayloadSizeThreshold = cfg.PayloadSizeThreshold

	switch cfg.Type {
	case StorageNone:
		return
	case StorageS3:
		driver, err := newS3Driver(ctx, cfg.S3)
		if err != nil {
			err = fmt.Errorf("error create s3 storage config: %w", err)
		}
		st.Drivers = []converter.StorageDriver{driver}
	default:
		err = fmt.Errorf("unsupported external storage type: %q", cfg.Type)
	}
	return
}
