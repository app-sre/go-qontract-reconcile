package rbac

import (
	"context"

	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/app-sre/go-qontract-reconcile/pkg/vault"
	"github.com/spf13/viper"
)

var IntegrationName = "advanced-cluster-security-rbac"

type getAcsOidcPermissionsFunc func(ctx context.Context) (*GetAcsOidcPermissionsResponse, error)

// AcsRbac is tasked with reconciling Red Hat Advanced Cluster Security RBAC resources
// https://docs.openshift.com/acs/4.2/operating/manage-user-access/manage-role-based-access-control-3630.html
type AcsRbac struct {
	config *AcsRbacConfig
	vc     *vault.Client

	// Used for mocking
	oidcPermissionsFunc getAcsOidcPermissionsFunc
}

// AcsRbacConfig is used to unmarshal yaml configuration for acs-related oidc permissions
type AcsRbacConfig struct {
	Endpoint string
	Token    string
}

func newAcsRbacConfig() *AcsRbacConfig {
	var cfg AcsRbacConfig
	sub := util.EnsureViperSub(viper.GetViper(), "advanced_cluster_security")
	sub.BindEnv("endpoint", "ACS_ENDPOINT")
	sub.BindEnv("token", "ACS_API_TOKEN")
	if err := sub.Unmarshal(&cfg); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &cfg
}

// NewAcsRbac creates a new AcsRbac integration struct
func NewAcsRbac() *AcsRbac {
	acsRbac := AcsRbac{
		config: newAcsRbacConfig(),
		oidcPermissionsFunc: func(ctx context.Context) (*GetAcsOidcPermissionsResponse, error) {
			return GetAcsOidcPermissions(ctx)
		},
	}
	return &acsRbac
}

func (a *AcsRbac) Setup(ctx context.Context) error { return nil }

func (a *AcsRbac) CurrentState(ctx context.Context, ri *reconcile.ResourceInventory) error {
	return nil
}

func (a *AcsRbac) DesiredState(ctx context.Context, ri *reconcile.ResourceInventory) error {
	return nil
}

func (a *AcsRbac) Reconcile(ctx context.Context, ri *reconcile.ResourceInventory) error {
	return nil
}

func (a *AcsRbac) LogDiff(ri *reconcile.ResourceInventory) {
}
