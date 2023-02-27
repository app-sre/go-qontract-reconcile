package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/app-sre/go-qontract-reconcile/pkg/aws"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

type Persistence interface {
	Exists(context.Context, string) (error, bool)
	Add(context.Context, string, interface{}) error
	Rm(context.Context, string) error
	Get(context.Context, string, interface{}) error
}

var _ Persistence = &S3State{}

type S3State struct {
	state     map[string]interface{}
	base_path string
	infix     string
	config    s3StateConfig
	client    aws.Client
}

type s3StateConfig struct {
	Bucket string
}

func newS3StateConfig() *s3StateConfig {
	var s3c s3StateConfig
	sub := util.EnsureViperSub(viper.GetViper(), "state_s3")
	sub.BindEnv("bucket", "APP_INTERFACE_STATE_BUCKET")
	if err := sub.Unmarshal(&s3c); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &s3c
}

func NewS3State(ctx context.Context, base_path, infix string, client aws.Client) *S3State {
	config := *newS3StateConfig()
	state := &S3State{
		state:     make(map[string]interface{}),
		base_path: base_path,
		infix:     infix,
		client:    client,
		config:    config,
	}
	return state
}

func (s *S3State) keyPath(key string) *string {
	return util.StrPointer(fmt.Sprintf("%s/%s/%s", s.base_path, s.infix, key))
}

func (s *S3State) Exists(ctx context.Context, key string) (error, bool) {
	util.Log().Debugw("Check key existsence in bucket", "key", s.keyPath(key), "bucket", s.config.Bucket)
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.config.Bucket,
		Key:    s.keyPath(key),
	})
	if err != nil {
		if strings.Contains(err.Error(), "api error NotFound: Not Found") {
			return nil, false
		}
		return err, false
	}
	return nil, true
}

func (s *S3State) Add(ctx context.Context, key string, value interface{}) error {
	util.Log().Debugw("Putting key to bucket", "key", s.keyPath(key), "bucket", s.config.Bucket)
	bytesOut, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.config.Bucket,
		Key:         s.keyPath(key),
		ContentType: util.StrPointer("application/json"),
		Body:        bytes.NewReader(bytesOut),
	})
	return err
}

func (s *S3State) Get(ctx context.Context, key string, value interface{}) error {
	util.Log().Debugw("Getting key from bucket", "key", s.keyPath(key), "bucket", s.config.Bucket)
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket:              &s.config.Bucket,
		Key:                 s.keyPath(key),
		ResponseContentType: util.StrPointer("application/json"),
	})
	if err != nil {
		return err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(bodyBytes, value)
}

func (s *S3State) Rm(ctx context.Context, key string) error {
	util.Log().Debugw("Deleting key from bucket", "key", s.keyPath(key), "bucket", s.config.Bucket)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.config.Bucket,
		Key:    s.keyPath(key),
	})
	if err != nil {
		return err
	}
	return nil
}
