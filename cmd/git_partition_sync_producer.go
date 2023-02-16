package cmd

import (
	"github.com/app-sre/go-qontract-reconcile/internal/gitpartitionsync/producer"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
)

func gitPartitionSyncProducer() {
	p := producer.NewGitPartitionSyncProducer()
	runner := reconcile.NewIntegrationRunner(p, "git-partition-sync-producer")
	runner.Run()
}
