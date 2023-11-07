// Package uservalidator contains code used by the user-validator
package keyvalidator

import (
	"context"
	"os"

	"github.com/app-sre/go-qontract-reconcile/pkg/pgp"
	"github.com/app-sre/go-qontract-reconcile/pkg/reconcile"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// KeyValidator is a Validation as described in github.com/app-sre/go-qontract-reconcile/pkg/integration.go
type KeyValidator struct {
	KeyValidatorConfig *KeyValidatorConfig
}

// KeyValidatorConfig is used to unmarshal yaml configuration for the user validator
type KeyValidatorConfig struct {
	Userfile string
}

func newKeyValidatorConfig() *KeyValidatorConfig {
	var vuc KeyValidatorConfig
	sub := util.EnsureViperSub(viper.GetViper(), "key_validator")
	sub.BindEnv("userfile", "KEY_VALIDATOR_USERFILE")
	if err := sub.Unmarshal(&vuc); err != nil {
		util.Log().Fatalw("Error while unmarshalling configuration %s", err.Error())
	}
	return &vuc
}

// NewKeyValidator Create a new KeyValidator integration struct
func NewKeyValidator() *KeyValidator {
	KeyValidator := KeyValidator{
		KeyValidatorConfig: newKeyValidatorConfig(),
	}
	return &KeyValidator
}

// Setup runs setup for key validator
func (i *KeyValidator) Setup(ctx context.Context) error {
	return nil
}

type userV1 struct {
	Path               string `json:"path"`
	Name               string `json:"name"`
	Org_username       string `json:"org_username"`
	Github_username    string `json:"github_username"`
	Slack_username     string `json:"slack_username"`
	Pagerduty_username string `json:"pagerduty_username"`
	Public_gpg_key     string `json:"public_gpg_key"`
}

// Validate run user validation
func (i *KeyValidator) Validate(ctx context.Context) ([]reconcile.ValidationError, error) {
	validationErrors := make([]reconcile.ValidationError, 0)

	userfile, err := os.ReadFile(i.KeyValidatorConfig.Userfile)
	if err != nil {
		return nil, err
	}

	user := userV1{}
	yaml.Unmarshal(userfile, &user)

	pgpKey := user.Public_gpg_key
	if len(pgpKey) > 0 {
		path := user.Path
		entity, err := pgp.DecodePgpKey(pgpKey, path)
		if err != nil {
			validationErrors = append(validationErrors, reconcile.ValidationError{
				Path:       path,
				Validation: "validatePgpKeys",
				Error:      err,
			})
			return validationErrors, nil
		}
		err = pgp.TestEncrypt(entity)
		if err != nil {
			validationErrors = append(validationErrors, reconcile.ValidationError{
				Path:       user.Path,
				Validation: "validatePgpKeys",
				Error:      err,
			})
		}
	}
	if len(validationErrors) == 0 {
		util.Log().Infof("Key provided for user %s is valid", user.Org_username)
	}

	return validationErrors, nil
}
