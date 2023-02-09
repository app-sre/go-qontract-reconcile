package producer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/app-sre/go-qontract-reconcile/pkg/aws/mock"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/xanzy/go-gitlab"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var GH_TEST_TOKEN = "gh_test_token"

func createTestProducer(awsClientMock *mock.MockClient, ghUrl string) *GitPartitionSyncProducer {
	c, _ := gitlab.NewClient(GH_TEST_TOKEN, gitlab.WithBaseURL(ghUrl))

	return &GitPartitionSyncProducer{
		config:    gitPartitionSyncProducerConfig{},
		glClient:  c,
		awsClient: awsClientMock,
	}
}

func setupGitlabMock(t *testing.T) *httptest.Server {
	return util.NewHttpTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/api/v4/projects/test%2Fproject/repository/commits/main" && r.Method == "GET" {
			fmt.Fprintf(w, `{"id": "test_sha"}`)
		}
	})
}

func TestCurrentStateError(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mock.NewMockClient(ctrl)

	mockClient.EXPECT().ListObjectsV2(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("api error NotFound: Not Found")).MaxTimes(1).MinTimes(1)

	producer := createTestProducer(mockClient, "")

	ri := reconcile.NewResourceInventory()

	err := producer.CurrentState(ctx, ri)

	assert.Error(t, err)
}

func TestCurrentStateBrokenKey(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mock.NewMockClient(ctrl)

	mockClient.EXPECT().ListObjectsV2(gomock.Any(), gomock.Any()).Return(&s3.ListObjectsV2Output{
		Contents: []types.Object{
			{
				Key: util.StrPointer("foo_bar"),
			},
		}}, nil).MaxTimes(1).MinTimes(1)

	producer := createTestProducer(mockClient, "")

	ri := reconcile.NewResourceInventory()

	err := producer.CurrentState(ctx, ri)

	assert.Error(t, err)
}

func TestCurrentStateOkay(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mock.NewMockClient(ctrl)

	mockClient.EXPECT().ListObjectsV2(gomock.Any(), gomock.Any()).Return(&s3.ListObjectsV2Output{
		Contents: []types.Object{
			{
				Key: util.StrPointer("eyJncm91cCI6InRlc3QiLCJwcm9qZWN0X25hbWUiOiJwcm9qZWN0IiwiY29tbWl0X3NoYSI6ImEiLCJsb2NhbF9icmFuY2giOiJtYWluIiwicmVtb3RlX2JyYW5jaCI6Im1haW4ifQo=.tar.age"),
			},
			{
				Key: util.StrPointer("eyJncm91cCI6InRlc3QiLCJwcm9qZWN0X25hbWUiOiJmb29iYXIiLCJjb21taXRfc2hhIjoiYSIsImxvY2FsX2JyYW5jaCI6Im1haW4iLCJyZW1vdGVfYnJhbmNoIjoibWFpbiJ9Cg==.tar.age"),
			},
			{
				Key: util.StrPointer("eyJncm91cCI6InRlc3QiLCJwcm9qZWN0X25hbWUiOiJmb29iYXIiLCJjb21taXRfc2hhIjoiYiIsImxvY2FsX2JyYW5jaCI6Im1haW4iLCJyZW1vdGVfYnJhbmNoIjoibWFpbiJ9Cg==.tar.age"),
			},
		}}, nil).MaxTimes(1).MinTimes(1)

	producer := createTestProducer(mockClient, "")

	ri := reconcile.NewResourceInventory()

	err := producer.CurrentState(ctx, ri)
	assert.NoError(t, err)

	current := ri.GetResourceState("test/project")
	assert.Equal(t, "a", current.Current.(*CurrentState).S3ObjectInfos[0].CommitSHA)
	assert.Equal(t, util.StrPointer("eyJncm91cCI6InRlc3QiLCJwcm9qZWN0X25hbWUiOiJwcm9qZWN0IiwiY29tbWl0X3NoYSI6ImEiLCJsb2NhbF9icmFuY2giOiJtYWluIiwicmVtb3RlX2JyYW5jaCI6Im1haW4ifQo=.tar.age"), current.Current.(*CurrentState).S3ObjectInfos[0].Key)
	assert.Len(t, current.Current.(*CurrentState).S3ObjectInfos, 1)

	current = ri.GetResourceState("test/foobar")
	assert.Len(t, current.Current.(*CurrentState).S3ObjectInfos, 2)
}

func TestDesiredState(t *testing.T) {
	ctx := context.Background()
	ri := reconcile.NewResourceInventory()

	glMock := setupGitlabMock(t)

	producer := createTestProducer(nil, glMock.URL)
	producer.getGitlabSyncAppsFunc = func(ctx context.Context) (*GetGitlabSyncAppsResponse, error) {
		return &GetGitlabSyncAppsResponse{
			Apps_v1: []GetGitlabSyncAppsApps_v1App_v1{
				{
					CodeComponents: []GetGitlabSyncAppsApps_v1App_v1CodeComponentsAppCodeComponents_v1{
						{GitlabSync: GetGitlabSyncAppsApps_v1App_v1CodeComponentsAppCodeComponents_v1GitlabSyncCodeComponentGitlabSync_v1{
							SourceProject: GetGitlabSyncAppsApps_v1App_v1CodeComponentsAppCodeComponents_v1GitlabSyncCodeComponentGitlabSync_v1SourceProjectCodeComponentGitlabSyncProject_v1{
								Name:   "project",
								Group:  "test",
								Branch: "main",
							},
							DestinationProject: GetGitlabSyncAppsApps_v1App_v1CodeComponentsAppCodeComponents_v1GitlabSyncCodeComponentGitlabSync_v1DestinationProjectCodeComponentGitlabSyncProject_v1{
								Name:   "project",
								Group:  "test",
								Branch: "foo",
							},
						}},
					},
				},
			},
		}, nil
	}
	err := producer.DesiredState(ctx, ri)
	assert.NoError(t, err)

	state := ri.GetResourceState("test/project")
	assert.NotNil(t, state.Desired)
	assert.Equal(t, "test_sha", state.Desired.(*S3ObjectInfo).CommitSHA)
}
