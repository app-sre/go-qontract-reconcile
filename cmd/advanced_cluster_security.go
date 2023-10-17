package cmd

import (
	"github.com/app-sre/go-qontract-reconcile/internal/acs/rbac"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
)

func advancedClusterSecurityRbac() {
	a := rbac.NewAcsRbac()
	runner := reconcile.NewIntegrationRunner(a, rbac.IntegrationName)
	runner.Run()
}
