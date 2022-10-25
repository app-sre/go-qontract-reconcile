package internal

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/app-sre/user-validator/internal/queries"
	. "github.com/app-sre/user-validator/pkg"
	"github.com/spf13/viper"
)

type githubValidateFunc func(ctx context.Context, user queries.UsersUsers_v1User_v1) *ValidationError

// ValidateUser is a Validationa s described in github.com/app-sre/user-validator/pkg/integration.go
type ValidateUser struct {
	AuthenticatedGithubClient *AuthenticatedGithubClient
	Vc                        *VaultClient
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
	sub := EnsureViperSub(viper.GetViper(), "user_validator")
	sub.SetDefault("concurrency", 10)
	sub.BindEnv("concurrency", "USER_VALIDATOR_CONCURRENCY")
	sub.BindEnv("invalidusers", "USER_VALIDATOR_INVALID_USERS")
	if err := sub.Unmarshal(&vuc); err != nil {
		Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
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
	orgs, err := queries.GithubOrgs(ctx)
	if err != nil {
		return err
	}
	i.Vc, err = NewVaultClient()
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
	i.AuthenticatedGithubClient, err = NewAuthenticatedGithubClient(ctx, secret.Data[tokenField].(string))
	if err != nil {
		return err
	}
	return nil
}

func (i *ValidateUser) validatePgpKeys(users queries.UsersResponse) []ValidationError {
	validUsers := i.removeInvalidUsers(&users)

	validationErrors := make([]ValidationError, 0)
	for _, user := range validUsers.GetUsers_v1() {
		pgpKey := user.GetPublic_gpg_key()
		if len(pgpKey) > 0 {
			path := user.GetPath()
			entity, err := DecodePgpKey(pgpKey, path)
			if err != nil {
				validationErrors = append(validationErrors, ValidationError{
					Path:       path,
					Validation: "validatePgpKeys",
					Error:      err,
				})
				continue
			}
			err = TestEncrypt(entity)
			if err != nil {
				validationErrors = append(validationErrors, ValidationError{
					Path:       user.GetPath(),
					Validation: "validatePgpKeys",
					Error:      err,
				})
			}
		}
	}
	return validationErrors
}

func (i *ValidateUser) validateUsersSinglePath(users queries.UsersResponse) []ValidationError {
	validationErrors := make([]ValidationError, 0)
	usersPaths := make(map[string][]string)

	for _, u := range users.GetUsers_v1() {
		if usersPaths[u.GetOrg_username()] == nil {
			usersPaths[u.GetOrg_username()] = make([]string, 0)
		}
		usersPaths[u.GetOrg_username()] = append(usersPaths[u.GetOrg_username()], u.GetPath())
	}

	for k, v := range usersPaths {
		if len(v) > 1 {
			for _, path := range v {
				validationErrors = append(validationErrors, ValidationError{
					Path:       path,
					Validation: "validateUsersSinglePath",
					Error:      fmt.Errorf("user \"%s\" has multiple user files", k),
				})
			}
		}
	}
	return validationErrors
}

func (i *ValidateUser) getAndValidateUser(ctx context.Context, user queries.UsersUsers_v1User_v1) *ValidationError {
	Log().Debugw("Getting github user", "user", user.GetOrg_username())
	ghUser, err := i.AuthenticatedGithubClient.GetUsers(ctx, user.GetGithub_username())
	if err != nil {
		Log().Debugw("API error", "user", user.Org_username, "error", err.Error())
		return &ValidationError{
			Path:       user.Path,
			Validation: "validateUsersGithub",
			Error:      err,
		}
	} else if ghUser.GetLogin() != user.GetGithub_username() {
		return &ValidationError{
			Path:       user.Path,
			Validation: "validateUsersGithub",
			Error: fmt.Errorf("Github username is case sensitive in OSD. GithubUsername \"%s\","+
				" configured Username \"%s\"", ghUser.GetLogin(), user.GetGithub_username()),
		}
	}
	return nil
}

func (i *ValidateUser) validateUsersGithub(ctx context.Context, users queries.UsersResponse) []ValidationError {
	validationErrors := make([]ValidationError, 0)
	validateWg := sync.WaitGroup{}
	gatherWg := sync.WaitGroup{}

	userChan := make(chan queries.UsersUsers_v1User_v1)
	retChan := make(chan ValidationError)

	gatherWg.Add(1)
	go func() {
		defer gatherWg.Done()
		for v := range retChan {
			validationErrors = append(validationErrors, v)
		}
	}()

	Log().Debugw("Starting github coroutines", "count", i.ValidateUserConfig.Concurrency)
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
		for _, user := range users.GetUsers_v1() {
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
func (i *ValidateUser) removeInvalidUsers(users *queries.UsersResponse) *queries.UsersResponse {
	returnUsers := &queries.UsersResponse{
		Users_v1: make([]queries.UsersUsers_v1User_v1, 0),
	}

	invalidPaths := make(map[string]bool)
	for _, user := range strings.Split(i.ValidateUserConfig.InvalidUsers, ",") {
		invalidPaths[user] = true
	}

	for _, user := range users.GetUsers_v1() {
		if _, ok := invalidPaths[user.GetPath()]; !ok {
			returnUsers.Users_v1 = append(returnUsers.GetUsers_v1(), user)
		} else {
			Log().Debugw("Skipping invalid user key", "path", user.GetPath())
		}
	}
	return returnUsers
}

// Validate run user validation
func (i *ValidateUser) Validate(ctx context.Context) ([]ValidationError, error) {
	allValidationErrors := make([]ValidationError, 0)
	users, err := queries.Users(ctx)
	if err != nil {
		return nil, err
	}

	allValidationErrors = ConcatValidationErrors(allValidationErrors, i.validateUsersSinglePath(*users))
	allValidationErrors = ConcatValidationErrors(allValidationErrors, i.validatePgpKeys(*users))
	allValidationErrors = ConcatValidationErrors(allValidationErrors, i.validateUsersGithub(ctx, *users))

	return allValidationErrors, nil
}
