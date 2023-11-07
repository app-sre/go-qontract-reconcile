// Package keyvalidator contains code used by the key-validator
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
	KeyValidatorConfig *Config
}

// Config is used to unmarshal yaml configuration for the key validator
type Config struct {
	Userfile string
}

func newKeyValidatorConfig() *Config {
	var vuc Config
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
	Path        string `json:"path"`
	OrgUsername string `json:"org_username"`
	GpgKey      string `json:"public_gpg_key"`
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

	pgpKey := user.GpgKey
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
		util.Log().Infof("Key provided for user %s is valid", user.OrgUsername)
	}

	return validationErrors, nil
}
