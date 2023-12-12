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
func (i *KeyValidator) Setup(_ context.Context) error {
	return nil
}

type userV1 struct {
	OrgUsername string `yaml:"org_username"`
	GpgKey      string `yaml:"public_gpg_key"`
}

// Validate run user validation
func (i *KeyValidator) Validate(_ context.Context) ([]reconcile.ValidationError, error) {
	userfile, err := os.ReadFile(i.KeyValidatorConfig.Userfile)
	if err != nil {
		return nil, err
	}
	var user userV1
	err = yaml.Unmarshal(userfile, &user)
	if err != nil {
		return nil, err
	}

	pgpKey := user.GpgKey

	if len(pgpKey) == 0 {
		util.Log().Infof("Key for user %s not provided", user.OrgUsername)
		return []reconcile.ValidationError{}, nil
	}
	entity, err := pgp.DecodePgpKey(pgpKey)
	if err != nil {
		return []reconcile.ValidationError{{
			Path:       i.KeyValidatorConfig.Userfile,
			Validation: "validatePgpKeys",
			Error:      err,
		}}, nil
	}
	err = pgp.TestEncrypt(entity)
	if err != nil {
		return []reconcile.ValidationError{{
			Path:       i.KeyValidatorConfig.Userfile,
			Validation: "validatePgpKeys",
			Error:      err,
		}}, nil
	}

	util.Log().Infof("Key provided for user %s is valid", user.OrgUsername)
	return []reconcile.ValidationError{}, nil
}
