// Package producer contains the producer integration for the git partition sync
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

type getGitlabSyncAppsFunc func(context.Context) (*GetGitlabSyncAppsResponse, error)

type gitPartitionSyncProducerConfig struct {
	AwsRegion  string
	Bucket     string
	GlBaseURL  string
	GlUsername string
	GlToken    string
	PublicKey  string
	Workdir    string
}

// GitPartitionSyncProducer is the producer integration for the git partition sync
type GitPartitionSyncProducer struct {
	config gitPartitionSyncProducerConfig

	glClient  *gitlab.Client
	awsClient aws.Client

	getGitlabSyncAppsFunc getGitlabSyncAppsFunc
}

type currentState struct {
	S3ObjectInfos []s3ObjectInfo
}

type s3ObjectInfo struct {
	Key       *string
	CommitSHA string
}

type decodedKey struct {
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

// NewGitPartitionSyncProducer returns a new GitPartitionSyncProducer
func NewGitPartitionSyncProducer() *GitPartitionSyncProducer {
	return &GitPartitionSyncProducer{
		config: *newNewGitPartitionSyncProducerConfig(),
		getGitlabSyncAppsFunc: func(ctx context.Context) (*GetGitlabSyncAppsResponse, error) {
			return GetGitlabSyncApps(ctx)
		},
	}
}

// Setup required directories and clients for the producer integration
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

// CurrentState gets all the currently synced repos from s3 and adds them as currentState to the ResourceInventory
func (g *GitPartitionSyncProducer) CurrentState(ctx context.Context, ri *reconcile.ResourceInventory) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	res, err := g.awsClient.ListObjectsV2(ctxTimeout, &s3.ListObjectsV2Input{
		Bucket: &g.config.Bucket,
	})
	if err != nil {
		return errors.Wrap(err, "error listing objects in s3")
	}

	var commitShas = make(map[string][]s3ObjectInfo)

	for _, obj := range res.Contents {
		// remove file extension before attempting decode
		// extension is .tar.age, split at first occurrence of .
		encodedKey := strings.SplitN(*obj.Key, ".", 2)[0]
		decodedBytes, err := base64.StdEncoding.DecodeString(encodedKey)
		if err != nil {
			return errors.Wrap(err, "error decoding key")
		}
		var jsonKey decodedKey
		err = json.Unmarshal(decodedBytes, &jsonKey)
		if err != nil {
			return errors.Wrap(err, "error unmarshalling json key")
		}
		targetPid := fmt.Sprintf("%s/%s", jsonKey.Group, jsonKey.ProjectName)
		if _, ok := commitShas[targetPid]; !ok {
			commitShas[targetPid] = []s3ObjectInfo{}
		}
		commitShas[targetPid] = append(commitShas[targetPid], s3ObjectInfo{
			Key:       obj.Key,
			CommitSHA: jsonKey.CommitSHA,
		})
	}

	for pid, objectInfos := range commitShas {
		ri.AddResourceState(pid, &reconcile.ResourceState{
			Current: &currentState{
				S3ObjectInfos: objectInfos,
			},
		})
	}

	return nil
}

// DesiredState gets the current commitID from Gitlab and adds it to the ResourceInventory
func (g *GitPartitionSyncProducer) DesiredState(ctx context.Context, ri *reconcile.ResourceInventory) error {
	apps, err := g.getGitlabSyncAppsFunc(ctx)
	if err != nil {
		return errors.Wrap(err, "Error while getting gitlab sync apps")
	}

	for _, app := range apps.GetApps_v1() {
		for _, cc := range app.GetCodeComponents() {
			sync := cc.GetGitlabSync()
			if len(sync.GetDestinationProject().Group) != 0 {
				sourcePid := fmt.Sprintf("%s/%s", sync.GetSourceProject().Group, sync.GetSourceProject().Name)
				targetPid := fmt.Sprintf("%s/%s", sync.GetDestinationProject().Group, sync.GetDestinationProject().Name)
				commit, _, err := g.glClient.Commits.GetCommit(sourcePid, sync.SourceProject.Branch, nil)
				if err != nil {
					return errors.Wrap(err, "Error while getting commit")
				}
				state := ri.GetResourceState(targetPid)
				if state != nil {
					ri.AddResourceState(targetPid, &reconcile.ResourceState{
						Config:  sync,
						Current: state.Current,
						Desired: &s3ObjectInfo{
							CommitSHA: commit.ID,
						},
					})
				} else {
					ri.AddResourceState(targetPid, &reconcile.ResourceState{
						Config: sync,
						Desired: &s3ObjectInfo{
							CommitSHA: commit.ID,
						},
					})
				}
			}
		}

	}
	return nil
}

func needsUpdate(current *currentState, desired *s3ObjectInfo) bool {
	if current != nil && desired != nil {
		for _, s3ObjectInfo := range current.S3ObjectInfos {
			// Current commit (from desired) is already in S3, thus exit
			if s3ObjectInfo.CommitSHA == desired.CommitSHA {
				return false
			}
		}
	}
	return true
}

type syncConfig struct {
	SourceProjectName       string
	SourceProjectGroup      string
	SourceBranch            string
	DestinationProjectName  string
	DestinationProjectGroup string
	DestinationBranch       string
}

// Reconcile syncs the repositories to S3 that have changed since the last run
func (g *GitPartitionSyncProducer) Reconcile(ctx context.Context, ri *reconcile.ResourceInventory) error {
	defer g.clear()

	for targetPid := range ri.State {
		util.Log().Debugw("Reconciling target", "target", targetPid)
		state := ri.GetResourceState(targetPid)
		sync := state.Config.(GetGitlabSyncAppsApps_v1App_v1CodeComponentsAppCodeComponents_v1GitlabSyncCodeComponentGitlabSync_v1)
		syncConfig := syncConfig{
			SourceProjectName:       sync.SourceProject.Name,
			SourceProjectGroup:      sync.SourceProject.Group,
			SourceBranch:            sync.SourceProject.Branch,
			DestinationProjectName:  sync.DestinationProject.Name,
			DestinationProjectGroup: sync.DestinationProject.Group,
			DestinationBranch:       sync.DestinationProject.Branch,
		}
		var current *currentState
		var desired *s3ObjectInfo
		if state.Current != nil {
			current = state.Current.(*currentState)
		}
		if state.Desired != nil {
			desired = state.Desired.(*s3ObjectInfo)
		}
		if needsUpdate(current, desired) {
			util.Log().Infow("Updating repo", "repo", targetPid)

			util.Log().Debugw("Cloning repo", "repo", targetPid)
			repoPath, err := g.cloneRepos(syncConfig)
			if err != nil {
				return errors.Wrapf(err, "Error while cloning repo %s", targetPid)
			}

			util.Log().Debugw("Tarring repo", "repo", targetPid)
			tarPath, err := g.tarRepos(repoPath, syncConfig)
			if err != nil {
				return errors.Wrapf(err, "Error while tarring repo %s", targetPid)
			}

			util.Log().Debugw("Encrypting repo", "repo", targetPid)
			encryptPath, err := g.encryptRepoTars(tarPath, syncConfig)
			if err != nil {
				return errors.Wrapf(err, "Error while encrypting repo %s", targetPid)
			}

			util.Log().Debugw("Uploading repo", "repo", targetPid)
			err = g.uploadLatest(ctx, encryptPath, desired.CommitSHA, syncConfig)
			if err != nil {
				return errors.Wrapf(err, "Error while uploading repo %s", targetPid)
			}
		}

		// Current is not nil means, there are old objects in S3, thus we need to check if objects need to be removed
		if current != nil {
			for _, s3ObjectInfo := range current.S3ObjectInfos {
				if s3ObjectInfo.CommitSHA != desired.CommitSHA {
					util.Log().Debugw("Removing outdated s3 object", "s3ObjectInfo", s3ObjectInfo)
					err := g.removeOutdated(ctx, s3ObjectInfo.Key)
					if err != nil {
						util.Log().Info("Deleting outdated s3 object")
						return errors.Wrap(err, "Error while removing outdated s3 object")
					}
				}
			}
		}
	}
	return nil
}

// LogDiff logs the diff between the current and desired state
func (g *GitPartitionSyncProducer) LogDiff(ri *reconcile.ResourceInventory) {
	for target := range ri.State {
		state := ri.GetResourceState(target)
		var current *currentState
		var desired *s3ObjectInfo

		if state.Current != nil {
			current = state.Current.(*currentState)
		}
		if state.Desired != nil {
			desired = state.Desired.(*s3ObjectInfo)
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
	cmd := exec.Command("rm", "-rf", encryptDirectory, tarDirectory, cloneDirectory)
	cmd.Dir = g.config.Workdir
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
