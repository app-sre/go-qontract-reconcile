package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/app-sre/go-qontract-reconcile/pkg/vault"
	"github.com/pkg/errors"
)

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

func getCredentialsFromEnv() *Credentials {
	if len(os.Getenv("AWS_ACCESS_KEY_ID")) != 0 && len(os.Getenv("AWS_SECRET_ACCESS_KEY")) != 0 {
		return &Credentials{
			AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		}
	}
	return nil
}

func getCredentialsFromVault(ctx context.Context, vc *vault.VaultClient, accountResponse *getAccountsResponse) (*Credentials, error) {
	accounts := accountResponse.GetAwsaccounts_v1()
	if len(accounts) != 1 {
		return nil, fmt.Errorf("expected one AWS account, got %d", len(accounts))
	}

	secret, err := vc.ReadSecret(accounts[0].AutomationToken.GetPath())

	if err != nil {
		return nil, errors.Wrap(err, "Error reading automation token")
	}
	aws_access_key_id := secret.Data["aws_access_key_id"].(string)
	aws_secret_access_key := secret.Data["aws_secret_access_key"].(string)

	return &Credentials{
		AccessKeyID:     aws_access_key_id,
		SecretAccessKey: aws_secret_access_key,
	}, nil

}

func GetAwsCredentials(ctx context.Context, vc *vault.VaultClient) (*Credentials, error) {
	secretsFromEnv := getCredentialsFromEnv()
	if secretsFromEnv != nil {
		return secretsFromEnv, nil
	}

	account := guessAccountName()
	if len(account) == 0 {
		return nil, fmt.Errorf("could not guess AWS account name")
	}
	accounts, err := getAccounts(ctx, account)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting AWS account info")
	}
	return getCredentialsFromVault(ctx, vc, accounts)
}

func guessAccountName() string {
	// qontract reconcile uses APP_INTERFACE_STATE_BUCKET_ACCOUNT for the account name of the state bucket
	if len(os.Getenv("APP_INTERFACE_STATE_BUCKET_ACCOUNT")) != 0 {
		return os.Getenv("APP_INTERFACE_STATE_BUCKET_ACCOUNT")
	}
	return ""
}
