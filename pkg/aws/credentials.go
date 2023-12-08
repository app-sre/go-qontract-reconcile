package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/app-sre/go-qontract-reconcile/pkg/vault"
	"github.com/pkg/errors"
)

// Credentials holds the AWS credentials that can be used with awsclient.NewClient
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	DefaultRegion   string
}

func getCredentialsFromEnv() *Credentials {
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" && os.Getenv("AWS_REGION") != "" {
		return &Credentials{
			AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			DefaultRegion:   os.Getenv("AWS_REGION"),
		}
	}
	return nil
}

func getCredentialsFromVault(vc *vault.Client, accountResponse *getAccountsResponse) (*Credentials, error) {
	accounts := accountResponse.GetAwsaccounts_v1()
	if len(accounts) != 1 {
		return nil, fmt.Errorf("expected one AWS account, got %d", len(accounts))
	}

	secret, err := vc.ReadSecret(accounts[0].AutomationToken.GetPath())

	if err != nil {
		return nil, errors.Wrap(err, "Error reading automation token")
	}
	awsAccessKeyID := secret.Data["aws_access_key_id"].(string)
	awsSecretAccessKey := secret.Data["aws_secret_access_key"].(string)

	return &Credentials{
		AccessKeyID:     awsAccessKeyID,
		SecretAccessKey: awsSecretAccessKey,
		DefaultRegion:   accounts[0].GetResourcesDefaultRegion(),
	}, nil

}

// GetAwsCredentials returns AWS credentials from the environment or from vault
func GetAwsCredentials(ctx context.Context, vc *vault.Client) (*Credentials, error) {
	secretsFromEnv := getCredentialsFromEnv()
	if secretsFromEnv != nil {
		return secretsFromEnv, nil
	} else if vc == nil {
		return nil, fmt.Errorf("could not get AWS credentials from environment and vault client is not configured")
	}

	account := guessAccountName()
	if len(account) == 0 {
		return nil, fmt.Errorf("could not guess AWS account name")
	}
	accounts, err := getAccounts(ctx, account)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting AWS account info")
	}
	return getCredentialsFromVault(vc, accounts)
}

func guessAccountName() string {
	// qontract reconcile uses APP_INTERFACE_STATE_BUCKET_ACCOUNT for the account name of the state bucket
	return os.Getenv("APP_INTERFACE_STATE_BUCKET_ACCOUNT")
}
