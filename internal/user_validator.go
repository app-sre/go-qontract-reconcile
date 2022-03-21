package internal

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"github.com/janboll/user-validator/internal/queries"
	. "github.com/janboll/user-validator/pkg"
	"github.com/keybase/go-crypto/openpgp"
	"github.com/keybase/go-crypto/openpgp/packet"
	"github.com/spf13/viper"
)

type githubValidateFunc func(ctx context.Context, user queries.UsersUsers_v1User_v1) *ValidationError

// ValidateUser is a Validationa s described in github.com/janboll/user-validator/pkg/integration.go
type ValidateUser struct {
	QClient                   *QontractClient
	AuthenticatedGithubClient *AuthenticatedGithubClient
	Vc                        *VaultClient
	ValidateUserConfig        *ValidateUserConfig

	// Used for mocking
	githubValidateFunc githubValidateFunc
}

// ValidateUserConfig is used to unmarshal yaml configuration for the user validator
type ValidateUserConfig struct {
	Concurrency int
}

func newValidateUserConfig() *ValidateUserConfig {
	var vuc ValidateUserConfig
	sub := EnsureViperSub(viper.GetViper(), "user_validator")
	sub.SetDefault("concurrency", 10)
	sub.BindEnv("concurrency", "USER_VALIDATOR_CONCURRENCY")
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
	i.QClient = NewQontractClient()
	orgs, err := queries.GithubOrgs(ctx, i.QClient.Client)
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
	i.AuthenticatedGithubClient = NewAuthenticatedGithubClient(ctx, secret.Data[tokenField].(string))

	return nil
}

func decodePgpKey(pgpKey string) (*openpgp.Entity, error) {
	pgpKey = strings.TrimRight(pgpKey, " \n\r")
	if strings.Contains(pgpKey, " ") {
		return nil, fmt.Errorf("PGP key has spaces in it")
	}

	suffix := ""
	for i := 0; i < strings.Count(pgpKey, "="); i++ {
		suffix += "="
	}
	if !strings.HasSuffix(pgpKey, suffix) {
		return nil, fmt.Errorf("Equals signs are not add the end")
	}

	data, err := base64.StdEncoding.DecodeString(pgpKey)
	if err != nil {
		return nil, fmt.Errorf("error decoding given PGP key: %w", err)
	}

	entity, err := openpgp.ReadEntity(packet.NewReader(bytes.NewBuffer(data)))
	if err != nil {
		return nil, fmt.Errorf("error parsing given PGP key: %w", err)
	}

	return entity, nil
}

func testEncrypt(entity *openpgp.Entity) error {
	ctBuf := bytes.NewBuffer(nil)
	pt, e := openpgp.Encrypt(ctBuf, []*openpgp.Entity{entity}, nil, nil, nil)
	if e != nil {
		return fmt.Errorf("error setting up encryption for PGP message: %w", e)
	}
	_, e = pt.Write([]byte("Hello World"))
	if e != nil {
		return fmt.Errorf("error encrypting PGP message: %w", e)
	}
	e = pt.Close()
	if e != nil {
		return fmt.Errorf("error closing encryption Stream: %w", e)
	}
	return nil
}

func (i *ValidateUser) validatePgpKeys(users queries.UsersResponse) []ValidationError {
	validationErrors := make([]ValidationError, 0)
	for _, user := range users.GetUsers_v1() {
		pgpKey := user.GetPublic_gpg_key()
		if len(pgpKey) > 0 {
			entity, err := decodePgpKey(pgpKey)
			if err != nil {
				validationErrors = append(validationErrors, ValidationError{
					Path:       user.GetPath(),
					Validation: "validatePgpKeys",
					Error:      err,
				})
				continue
			}
			err = testEncrypt(entity)
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

// Validate run user validation
func (i *ValidateUser) Validate(ctx context.Context) ([]ValidationError, error) {
	allValidationErrors := make([]ValidationError, 0)
	users, err := queries.Users(ctx, i.QClient.Client)
	if err != nil {
		return nil, err
	}

	allValidationErrors = ConcatValidationErrors(allValidationErrors, i.validateUsersSinglePath(*users))
	allValidationErrors = ConcatValidationErrors(allValidationErrors, i.validatePgpKeys(*users))
	allValidationErrors = ConcatValidationErrors(allValidationErrors, i.validateUsersGithub(ctx, *users))

	return allValidationErrors, nil
}
