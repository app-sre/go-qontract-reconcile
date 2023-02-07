package producer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/app-sre/go-qontract-reconcile/pkg/aws"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/xanzy/go-gitlab"
)

type gitPartitionSyncProducerConfig struct {
	AwsRegion  string
	Bucket     string
	GlBaseURL  string
	GlUsername string
	GlToken    string
	PublicKey  string
	Workdir    string
}

type GitPartitionSyncProducer struct {
	config gitPartitionSyncProducerConfig

	glClient  *gitlab.Client
	awsClient aws.Client
}

type S3ObjectInfo struct {
	Key       *string
	CommitSHA string
}

type DecodedKey struct {
	Group        string `json:"group"`
	ProjectName  string `json:"project_name"`
	CommitSHA    string `json:"commit_sha"`
	LocalBranch  string `json:"local_branch"`
	RemoteBranch string `json:"remote_branch"`
}

func newNewGitPartitionSyncProducerConfig() *gitPartitionSyncProducerConfig {
	var cfg gitPartitionSyncProducerConfig
	sub := util.EnsureViperSub(viper.GetViper(), "gitPartitionSyncProducer")
	sub.BindEnv("bucket", "AWS_GIT_SYNC_BUCKET")
	sub.BindEnv("glBaseURL", "GITLAB_BASE_URL")
	sub.BindEnv("glUsername", "GITLAB_USERNAME")
	sub.BindEnv("glToken", "GITLAB_TOKEN")
	sub.BindEnv("publicKey", "PUBLIC_KEY")
	sub.BindEnv("workdir", "WORKDIR")
	if err := sub.Unmarshal(&cfg); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &cfg
}

func NewGitPartitionSyncProducer() *GitPartitionSyncProducer {
	return &GitPartitionSyncProducer{
		config: *newNewGitPartitionSyncProducerConfig(),
	}
}

func (g *GitPartitionSyncProducer) CurrentState(ctx context.Context, ri *reconcile.ResourceInventory) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	res, err := g.awsClient.ListObjectsV2(ctxTimeout, &s3.ListObjectsV2Input{
		Bucket: &g.config.Bucket,
	})
	if err != nil {
		return errors.Wrap(err, "error listing objects in s3")
	}

	for _, obj := range res.Contents {
		// remove file extension before attempting decode
		// extension is .tar.age, split at first occurrence of .
		encodedKey := strings.SplitN(*obj.Key, ".", 2)[0]
		decodedBytes, err := base64.StdEncoding.DecodeString(encodedKey)
		if err != nil {
			return errors.Wrap(err, "error decoding key")
		}
		var jsonKey DecodedKey
		err = json.Unmarshal(decodedBytes, &jsonKey)
		if err != nil {
			return errors.Wrap(err, "error unmarshalling json key")
		}
		pid := fmt.Sprintf("%s/%s", jsonKey.Group, jsonKey.ProjectName)
		ri.AddResourceState(pid, &reconcile.ResourceState{
			Current: &S3ObjectInfo{
				Key:       obj.Key,
				CommitSHA: jsonKey.CommitSHA,
			}})
	}
	return nil
}

func (g *GitPartitionSyncProducer) DesiredState(ctx context.Context, ri *reconcile.ResourceInventory) error {
	apps, err := GetGitlabSyncApps(ctx)
	if err != nil {
		return errors.Wrap(err, "Error while getting gitlab sync apps")
	}

	for _, app := range apps.GetApps_v1() {
		for _, cc := range app.GetCodeComponents() {
			sync := cc.GetGitlabSync()
			if len(sync.GetDestinationProject().Group) != 0 {
				pid := fmt.Sprintf("%s/%s", sync.GetSourceProject().Group, sync.GetSourceProject().Name)
				commit, _, err := g.glClient.Commits.GetCommit(pid, sync.SourceProject.Branch, nil)
				if err != nil {
					return errors.Wrap(err, "Error while getting commit")
				}
				state := ri.GetResourceState(pid)
				if state != nil {
					rsNew := &reconcile.ResourceState{
						Config:  sync,
						Current: state.Current,
						Desired: &S3ObjectInfo{
							CommitSHA: commit.ID,
						},
					}
					ri.AddResourceState(pid, rsNew)
				} else {
					ri.AddResourceState(pid, &reconcile.ResourceState{
						Config: sync,
						Desired: &S3ObjectInfo{
							CommitSHA: commit.ID,
						},
					})
				}
			}
		}

	}
	return nil
}

func (g *GitPartitionSyncProducer) Setup(ctx context.Context) error {
	cmd := exec.Command("mkdir", "-p", g.config.Workdir)
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "Error while creating workdir")
	}

	gl, err := gitlab.NewClient(g.config.GlToken, gitlab.WithBaseURL(fmt.Sprintf("%s/api/v4", g.config.GlBaseURL)))
	if err != nil {
		return errors.Wrap(err, "Error while creating gitlab client")
	}
	g.glClient = gl

	awsSecrets, err := aws.GetAwsCredentials(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "Error getting AWS secrets")
	}

	awsclient, err := aws.NewClient(ctx, awsSecrets)
	if err != nil {
		return errors.Wrapf(err, "Error getting AWS client")
	}

	g.awsClient = awsclient

	return nil
}

func needsUpdate(current, desired *S3ObjectInfo) bool {
	if current == nil && desired != nil {
		return true
	} else if current != nil && desired != nil {
		if current.CommitSHA != desired.CommitSHA {
			return true
		}
	}
	return false
}

type syncConfig struct {
	SourceProjectName       string
	SourceProjectGroup      string
	SourceBranch            string
	DestinationProjectName  string
	DestinationProjectGroup string
	DestinationBranch       string
}

func (g *GitPartitionSyncProducer) Reconcile(ctx context.Context, ri *reconcile.ResourceInventory) error {
	for target := range ri.State {
		state := ri.GetResourceState(target)
		sync := state.Config.(GetGitlabSyncAppsApps_v1App_v1CodeComponentsAppCodeComponents_v1GitlabSyncCodeComponentGitlabSync_v1)
		syncConfig := syncConfig{
			SourceProjectName:       sync.SourceProject.Name,
			SourceProjectGroup:      sync.SourceProject.Group,
			SourceBranch:            sync.SourceProject.Branch,
			DestinationProjectName:  sync.DestinationProject.Name,
			DestinationProjectGroup: sync.DestinationProject.Group,
			DestinationBranch:       sync.DestinationProject.Branch,
		}
		var current, desired *S3ObjectInfo
		if state.Current != nil {
			current = state.Current.(*S3ObjectInfo)
		}
		if state.Desired != nil {
			desired = state.Desired.(*S3ObjectInfo)
		}
		if needsUpdate(current, desired) {
			util.Log().Info("Updating repo", "repo", target)

			util.Log().Debug("Cloning repo")
			repoPath, err := g.cloneRepos(syncConfig)
			if err != nil {
				return err
			}

			util.Log().Debug("Tarring repo")
			tarPath, err := g.tarRepos(repoPath, syncConfig)
			if err != nil {
				return err
			}

			util.Log().Debug("Encrypting repo")
			encryptPath, err := g.encryptRepoTars(tarPath, syncConfig)
			if err != nil {
				return err
			}

			util.Log().Debug("Uploading repo")
			err = g.uploadLatest(ctx, encryptPath, desired.CommitSHA, syncConfig)
			if err != nil {
				return err
			}

		} else if state.Current != nil && state.Desired == nil {
			err := g.removeOutdated(ctx, desired.Key)
			if err != nil {
				util.Log().Info("Deleting outdated s3 object")
				return errors.Wrap(err, "Error while removing outdated s3 object")
			}

		}
	}
	return nil
}

func (g *GitPartitionSyncProducer) LogDiff(ri *reconcile.ResourceInventory) {
	for target := range ri.State {
		state := ri.GetResourceState(target)
		var current, desired *S3ObjectInfo

		if state.Current != nil {
			current = state.Current.(*S3ObjectInfo)
		}
		if state.Desired != nil {
			desired = state.Desired.(*S3ObjectInfo)
		}
		if needsUpdate(current, desired) {
			util.Log().Infof("Resource %s has changed", target)
		}
	}
}

func (g *GitPartitionSyncProducer) clean(directory string) error {
	cmd := exec.Command("rm", "-rf", directory)
	cmd.Dir = g.config.Workdir
	err := cmd.Run()
	if err != nil {
		return err
	}
	cmd = exec.Command("mkdir", directory)
	cmd.Dir = g.config.Workdir
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// clear all items in working directory
func (g *GitPartitionSyncProducer) clear() error {
	cmd := exec.Command("rm", "-rf", ENCRYPT_DIRECTORY, TAR_DIRECTORY, CLONE_DIRECTORY)
	cmd.Dir = g.config.Workdir
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
