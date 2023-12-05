// Package uservalidator contains code used by the user-validator
package uservalidator

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/app-sre/go-qontract-reconcile/pkg/github"
	"github.com/app-sre/go-qontract-reconcile/pkg/gql"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/app-sre/go-qontract-reconcile/pkg/vault"
	"github.com/spf13/viper"
)

type githubValidateFunc func(ctx context.Context, user UsersUsers_v1User_v1) *reconcile.ValidationError

// ValidateUser is a Validationa s described in github.com/app-sre/go-qontract-reconcile/pkg/integration.go
type ValidateUser struct {
	AuthenticatedGithubClient *github.AuthenticatedGithubClient
	Vc                        *vault.Client
	ValidateUserConfig        *ValidateUserConfig

	// Used for mocking
	githubValidateFunc githubValidateFunc
}

// ValidateUserConfig is used to unmarshal yaml configuration for the user validator
type ValidateUserConfig struct {
	Concurrency  int
	InvalidUsers string
}

func newValidateUserConfig() *ValidateUserConfig {
	var vuc ValidateUserConfig
	sub := util.EnsureViperSub(viper.GetViper(), "user_validator")
	sub.SetDefault("concurrency", 10)
	sub.BindEnv("concurrency", "USER_VALIDATOR_CONCURRENCY")
	if err := sub.Unmarshal(&vuc); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &vuc
}

// NewValidateUser Create a new ValidateUser integration struct
func NewValidateUser() *ValidateUser {
	validateUser := ValidateUser{
		ValidateUserConfig: newValidateUserConfig(),
	}
	validateUser.githubValidateFunc = validateUser.getAndValidateUser
	return &validateUser
}

// Setup runs setup for user validator
func (i *ValidateUser) Setup(ctx context.Context) error {
	var err error
	orgs, err := GithubOrgs(ctx)
	if err != nil {
		return err
	}
	i.Vc, err = vault.NewVaultClient()
	if err != nil {
		return err
	}

	var tokenPath string
	var tokenField string
	for _, org := range orgs.GetGithuborg_v1() {
		if org.GetDefault() {
			tokenPath = org.GetToken().Path
			tokenField = org.GetToken().Field
		}
	}
	secret, err := i.Vc.ReadSecret(tokenPath)
	if err != nil {
		return err
	}
	if secret == nil {
		return fmt.Errorf("Github Secret \"%s\" not found", tokenPath)
	}
	i.AuthenticatedGithubClient, err = github.NewAuthenticatedGithubClient(ctx, secret.Data[tokenField].(string))
	if err != nil {
		return err
	}
	return nil
}

func (i *ValidateUser) validateUsersSinglePath(users []UsersUsers_v1User_v1) []reconcile.ValidationError {
	validationErrors := make([]reconcile.ValidationError, 0)
	usersPaths := make(map[string][]string)

	for _, u := range users {
		if usersPaths[u.GetOrg_username()] == nil {
			usersPaths[u.GetOrg_username()] = make([]string, 0)
		}
		usersPaths[u.GetOrg_username()] = append(usersPaths[u.GetOrg_username()], u.GetPath())
	}

	for k, v := range usersPaths {
		if len(v) > 1 {
			for _, path := range v {
				validationErrors = append(validationErrors, reconcile.ValidationError{
					Path:       path,
					Validation: "validateUsersSinglePath",
					Error:      fmt.Errorf("user \"%s\" has multiple user files", k),
				})
			}
		}
	}
	return validationErrors
}

func (i *ValidateUser) getAndValidateUser(ctx context.Context, user UsersUsers_v1User_v1) *reconcile.ValidationError {
	util.Log().Debugw("Getting github user", "user", user.GetOrg_username())
	ghUser, err := i.AuthenticatedGithubClient.GetUsers(ctx, user.GetGithub_username())
	if err != nil {
		util.Log().Debugw("API error", "user", user.Org_username, "error", err.Error())
		return &reconcile.ValidationError{
			Path:       user.Path,
			Validation: "validateUsersGithub",
			Error:      err,
		}
	} else if ghUser.GetLogin() != user.GetGithub_username() {
		return &reconcile.ValidationError{
			Path:       user.Path,
			Validation: "validateUsersGithub",
			Error: fmt.Errorf("Github username is case sensitive in OSD. GithubUsername \"%s\","+
				" configured Username \"%s\"", ghUser.GetLogin(), user.GetGithub_username()),
		}
	}
	return nil
}

func (i *ValidateUser) validateUsersGithub(ctx context.Context, users []UsersUsers_v1User_v1) []reconcile.ValidationError {
	validationErrors := make([]reconcile.ValidationError, 0)
	validateWg := sync.WaitGroup{}
	gatherWg := sync.WaitGroup{}

	userChan := make(chan UsersUsers_v1User_v1)
	retChan := make(chan reconcile.ValidationError)

	gatherWg.Add(1)
	go func() {
		defer gatherWg.Done()
		for v := range retChan {
			validationErrors = append(validationErrors, v)
		}
	}()

	util.Log().Debugw("Starting github coroutines", "count", i.ValidateUserConfig.Concurrency)
	for t := 0; t < i.ValidateUserConfig.Concurrency; t++ {
		validateWg.Add(1)
		go func() {
			defer validateWg.Done()
			for user := range userChan {
				validationError := i.githubValidateFunc(ctx, user)
				if validationError != nil {
					retChan <- *validationError
				}
			}
		}()
	}

	go func() {
		defer close(userChan)
		for _, user := range users {
			userChan <- user
		}
	}()

	validateWg.Wait()
	close(retChan)

	gatherWg.Wait()

	return validationErrors
}

// TODO: This is just a hack, really we should remove the invalid keys from app-interface
//
//	and mange invalid keys stateful
func (i *ValidateUser) removeInvalidUsers(users *UsersResponse) *UsersResponse {
	returnUsers := &UsersResponse{
		Users_v1: make([]UsersUsers_v1User_v1, 0),
	}

	invalidPaths := make(map[string]bool)
	for _, user := range strings.Split(i.ValidateUserConfig.InvalidUsers, ",") {
		invalidPaths[user] = true
	}

	for _, user := range users.GetUsers_v1() {
		if _, ok := invalidPaths[user.GetPath()]; !ok {
			returnUsers.Users_v1 = append(returnUsers.GetUsers_v1(), user)
		} else {
			util.Log().Debugw("Skipping invalid user key", "path", user.GetPath())
		}
	}
	return returnUsers
}

func findUsersToValidate(users *UsersResponse, compareUsers *UsersResponse) []UsersUsers_v1User_v1 {
	userMap := make(map[string]UsersUsers_v1User_v1)
	for _, user := range users.GetUsers_v1() {
		userMap[user.GetPath()] = user
	}
	fmt.Println(userMap)
	compareUserMap := make(map[string]UsersUsers_v1User_v1)
	for _, user := range compareUsers.GetUsers_v1() {
		compareUserMap[user.GetPath()] = user
	}
	fmt.Println(compareUserMap)

	var usersToValidate = make([]UsersUsers_v1User_v1, 0)

	for k, v := range userMap {
		if _, ok := compareUserMap[k]; ok {
			if !reflect.DeepEqual(v, compareUserMap[k]) {
				usersToValidate = append(usersToValidate, v)
			}
		} else {
			usersToValidate = append(usersToValidate, v)
		}
	}
	return usersToValidate
}

// Validate run user validation
func (i *ValidateUser) Validate(ctx context.Context) ([]reconcile.ValidationError, error) {
	allValidationErrors := make([]reconcile.ValidationError, 0)
	users, err := Users(ctx)
	if err != nil {
		return nil, err
	}

	compareUsers, err := Users(context.WithValue(ctx, gql.UseCompareClientKey, true))
	if err != nil {
		return nil, err
	}

	if users == nil || compareUsers == nil {
		return nil, fmt.Errorf("No users found")
	}

	usersToValidate := findUsersToValidate(users, compareUsers)

	allValidationErrors = reconcile.ConcatValidationErrors(allValidationErrors, i.validateUsersSinglePath(usersToValidate))
	allValidationErrors = reconcile.ConcatValidationErrors(allValidationErrors, i.validateUsersGithub(ctx, usersToValidate))

	return allValidationErrors, nil
}
