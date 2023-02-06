package producer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
)

// concurrently deletes objects from s3 sync bucket that are no longer needed
func (g *GitPartitionSyncProducer) removeOutdated(ctx context.Context, keyToDelete *string) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := g.s3Client.DeleteObject(ctxTimeout, &s3.DeleteObjectInput{
		Bucket: &g.config.Bucket,
		Key:    keyToDelete,
	})

	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to delete object %s", *keyToDelete))
	}

	return nil
}

// cocurrently uploads latest encrypted tars to target s3 bucket
func (g *GitPartitionSyncProducer) uploadLatest(ctx context.Context, encryptPath, dGroup, dName, commitSha, sBranch, dBranch string) error {
	ctxTimeout, cancel := context.WithCancel(ctx)
	defer cancel()

	jsonStruct := &DecodedKey{
		Group:        dGroup,
		ProjectName:  dName,
		CommitSHA:    commitSha,
		LocalBranch:  sBranch,
		RemoteBranch: dBranch,
	}

	jsonBytes, err := json.Marshal(jsonStruct)
	if err != nil {
		return err
	}

	encodedJsonStr := base64.StdEncoding.EncodeToString(jsonBytes)
	objKey := fmt.Sprintf("%s.tar.age", encodedJsonStr)

	f, err := os.Open(encryptPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = g.s3Client.PutObject(ctxTimeout, &s3.PutObjectInput{
		Bucket: &g.config.Bucket,
		Key:    &objKey,
		Body:   f,
	})

	if err != nil {
		return err
	}

	return nil
}
