// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package aws

import (
	"context"

	"github.com/Khan/genqlient/graphql"
	"github.com/app-sre/go-qontract-reconcile/pkg/gql"
)

// __getAccountsInput is used internally by genqlient
type __getAccountsInput struct {
	Name string `json:"name"`
}

// GetName returns __getAccountsInput.Name, and is useful for accessing the field via an interface.
func (v *__getAccountsInput) GetName() string { return v.Name }

// getAccountsAwsaccounts_v1AWSAccount_v1 includes the requested fields of the GraphQL type AWSAccount_v1.
type getAccountsAwsaccounts_v1AWSAccount_v1 struct {
	Name                   string                                                              `json:"name"`
	ResourcesDefaultRegion string                                                              `json:"resourcesDefaultRegion"`
	AutomationToken        getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1 `json:"automationToken"`
}

// GetName returns getAccountsAwsaccounts_v1AWSAccount_v1.Name, and is useful for accessing the field via an interface.
func (v *getAccountsAwsaccounts_v1AWSAccount_v1) GetName() string { return v.Name }

// GetResourcesDefaultRegion returns getAccountsAwsaccounts_v1AWSAccount_v1.ResourcesDefaultRegion, and is useful for accessing the field via an interface.
func (v *getAccountsAwsaccounts_v1AWSAccount_v1) GetResourcesDefaultRegion() string {
	return v.ResourcesDefaultRegion
}

// GetAutomationToken returns getAccountsAwsaccounts_v1AWSAccount_v1.AutomationToken, and is useful for accessing the field via an interface.
func (v *getAccountsAwsaccounts_v1AWSAccount_v1) GetAutomationToken() getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1 {
	return v.AutomationToken
}

// getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1 includes the requested fields of the GraphQL type VaultSecret_v1.
type getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1 struct {
	Path    string `json:"path"`
	Field   string `json:"field"`
	Version int    `json:"version"`
	Format  string `json:"format"`
}

// GetPath returns getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1.Path, and is useful for accessing the field via an interface.
func (v *getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1) GetPath() string {
	return v.Path
}

// GetField returns getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1.Field, and is useful for accessing the field via an interface.
func (v *getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1) GetField() string {
	return v.Field
}

// GetVersion returns getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1.Version, and is useful for accessing the field via an interface.
func (v *getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1) GetVersion() int {
	return v.Version
}

// GetFormat returns getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1.Format, and is useful for accessing the field via an interface.
func (v *getAccountsAwsaccounts_v1AWSAccount_v1AutomationTokenVaultSecret_v1) GetFormat() string {
	return v.Format
}

// getAccountsResponse is returned by getAccounts on success.
type getAccountsResponse struct {
	Awsaccounts_v1 []getAccountsAwsaccounts_v1AWSAccount_v1 `json:"awsaccounts_v1"`
}

// GetAwsaccounts_v1 returns getAccountsResponse.Awsaccounts_v1, and is useful for accessing the field via an interface.
func (v *getAccountsResponse) GetAwsaccounts_v1() []getAccountsAwsaccounts_v1AWSAccount_v1 {
	return v.Awsaccounts_v1
}

// The query or mutation executed by getAccounts.
const getAccounts_Operation = `
query getAccounts ($name: String) {
	awsaccounts_v1(name: $name) {
		name
		resourcesDefaultRegion
		automationToken {
			path
			field
			version
			format
		}
	}
}
`

func getAccounts(
	ctx_ context.Context,
	name string,
) (*getAccountsResponse, error) {
	req_ := &graphql.Request{
		OpName: "getAccounts",
		Query:  getAccounts_Operation,
		Variables: &__getAccountsInput{
			Name: name,
		},
	}
	var err_ error
	var client_ graphql.Client

	client_, err_ = gql.NewQontractClient(ctx_)
	if err_ != nil {
		return nil, err_
	}

	var data_ getAccountsResponse
	resp_ := &graphql.Response{Data: &data_}

	err_ = client_.MakeRequest(
		ctx_,
		req_,
		resp_,
	)

	return &data_, err_
}
