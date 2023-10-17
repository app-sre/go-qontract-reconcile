package rbac

import (
	"context"

	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/app-sre/go-qontract-reconcile/pkg/stackrox"
)

var IntegrationName = "advanced-cluster-security-rbac"

type getAcsOidcPermissionsFunc func(ctx context.Context) (*GetAcsOidcPermissionsResponse, error)

// AcsRbac is tasked with reconciling Red Hat Advanced Cluster Security RBAC resources
// https://docs.openshift.com/acs/4.2/operating/manage-user-access/manage-role-based-access-control-3630.html
type AcsRbac struct {
	sc *stackrox.StackroxClient

	// Used for mocking
	oidcPermissionsFunc getAcsOidcPermissionsFunc
}

// NewAcsRbac creates a new AcsRbac integration struct
func NewAcsRbac() *AcsRbac {
	acsRbac := AcsRbac{
		oidcPermissionsFunc: func(ctx context.Context) (*GetAcsOidcPermissionsResponse, error) {
			return GetAcsOidcPermissions(ctx)
		},
	}
	return &acsRbac
}

// Setup required clients for ACS rbac integration
func (a *AcsRbac) Setup(ctx context.Context) error {
	var err error
	a.sc, err = stackrox.NewClient()
	if err != nil {
		return err
	}
	return nil
}

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
