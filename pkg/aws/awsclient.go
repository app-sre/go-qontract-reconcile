// Package aws provides a mockable client for interacting with AWS.
// revive:disable:unexported-return
package aws

import (
	"context"

	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

//go:generate go run github.com/Khan/genqlient

var _ = `# @genqlient
query getAccounts($name: String) {
	awsaccounts_v1 (name: $name) {
		name
		resourcesDefaultRegion
		automationToken {
			path
			field
			version
			format
		}
	}
}
`

//go:generate mockgen -source=./awsclient.go -destination=./mock/zz_generated.mock_client.go -package=mock

// Client is a wrapper object for actual AWS SDK clients to allow for easier testing.
type Client interface {
	//S3
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type awsClient struct {
	s3Client s3.Client
}

func (c *awsClient) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return c.s3Client.GetObject(ctx, params, optFns...)
}

func (c *awsClient) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return c.s3Client.HeadObject(ctx, params, optFns...)
}

func (c *awsClient) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return c.s3Client.PutObject(ctx, params, optFns...)
}

func (c *awsClient) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return c.s3Client.DeleteObject(ctx, params, optFns...)
}

func (c *awsClient) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return c.s3Client.ListObjectsV2(ctx, params, optFns...)
}

type awsClientConfig struct {
	Region string
}

func newAwsClientConfig() *awsClientConfig {
	var cfg awsClientConfig
	sub := util.EnsureViperSub(viper.GetViper(), "aws")
	sub.BindEnv("region", "AWS_REGION")
	if err := sub.Unmarshal(&cfg); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &cfg
}

// NewClient returns a new AWS client, that implements the Client interface.
func NewClient(ctx context.Context, creds *Credentials) (*awsClient, error) {
	awsCfg := newAwsClientConfig()

	region := awsCfg.Region
	if region == "" {
		region = creds.DefaultRegion
	}

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, "")))
	if err != nil {
		return nil, errors.Wrap(err, "error creating AWS configuration")
	}

	return &awsClient{
		s3Client: *s3.NewFromConfig(cfg),
	}, nil
}
