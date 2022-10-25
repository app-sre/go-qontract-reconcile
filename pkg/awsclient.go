// inspired by https://github.com/openshift/aws-account-operator/blob/master/pkg/awsclient/client.go

package pkg

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

//go:generate mockgen -source=./awsclient.go -destination=./mock/zz_generated.mock_client.go -package=mock

// Client is a wrapper object for actual AWS SDK clients to allow for easier testing.
type Client interface {
	//S3
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
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

func newAwsConfig(ctx context.Context) *aws.Config {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		Log().Fatalw("Error creating AWS configuration %s", err.Error())
	}
	return &cfg
}

func NewClient(ctx context.Context) *awsClient {
	return &awsClient{
		s3Client: *s3.NewFromConfig(*newAwsConfig(ctx)),
	}
}
