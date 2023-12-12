package pgp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ProtonMail/gopenpgp/v2/constants"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
	"github.com/stretchr/testify/assert"
)

var (
	publicFile             = "../../test/data/public_key.b64"
	publicSingleLineFile   = "../../test/data/public_key.single-line.b64"
	publicNoEncryptionFile = "../../test/data/public_key.no-encryption.b64"
	privateFile            = "../../test/data/private_key.b64"
	expiredFile            = "../../test/data/expired_key.b64"
	eccFile                = "../../test/data/ecc_key.b64"
	armoredKeyFile         = "../../test/data/armored_key.b64"
)

func TestDecodePgpKeyFailDecode(t *testing.T) {
	entity, err := DecodePgpKey("a")
	assert.Nil(t, entity)
	assert.EqualError(t, err, "error decoding given PGP key: illegal base64 data at input byte 0")
}

func TestDecodePgpKeyFailEntity(t *testing.T) {
	entity, err := DecodePgpKey("Zm9vCg==")
	assert.Nil(t, entity)
	assert.EqualError(t, err, "error parsing given PGP key: openpgp: invalid data: tag byte does not have MSB set")
}

func TestDecodePgpKeyOkay(t *testing.T) {
	key := util.ReadKeyFile(t, publicFile)
	entity, err := DecodePgpKey(string(key))
	assert.Nil(t, err)
	assert.NotNil(t, entity)

	key = util.ReadKeyFile(t, publicSingleLineFile)
	entity, err = DecodePgpKey(string(key))
	assert.Nil(t, err)
	assert.NotNil(t, entity)
}

func TestTestDecodeEccOkay(t *testing.T) {
	key := util.ReadKeyFile(t, eccFile)
	entity, err := DecodePgpKey(string(key))
	assert.Nil(t, err)
	assert.NotNil(t, entity)
}

func TestDecodePgpKeyInvalidArmoredKey(t *testing.T) {
	_, err := DecodePgpKey("-----BEGIN PGP PUBLIC KEY BLOCK-----")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "please remove type headers")

	key := util.ReadKeyFile(t, publicFile)
	// The CRC24 checksum for this public key is: 15b421 (encoded as =FbQh),
	// validate if DecodePgpKey supports key with checksum
	keyData, err := DecodePgpKey(string(key) + "\n=FbQh\n")
	assert.Nil(t, err)
	assert.NotNil(t, keyData)
}

func TestDecodePgpKeyInvalidArmoredKeyChecksum(t *testing.T) {
	key := util.ReadKeyFile(t, publicFile)
	// The CRC24 checksum for this public key is: 15b421 (encoded as =FbQh).
	// add an invalid CRC at the end of the key.
	_, err := DecodePgpKey(string(key) + "\n=T3st\n")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "error decoding given ASCII-armored PGP key: openpgp: invalid data: armor invalid")
}

func TestDecodePgpKeyInvalidPrivateKey(t *testing.T) {
	key := util.ReadKeyFile(t, privateFile)
	_, err := DecodePgpKey(string(key))
	assert.NotNil(t, err)
	assert.EqualError(t, err, "given PGP key is not a Public Key")
}

func TestDecodePgpKeyInvalidSpaces(t *testing.T) {
	_, err := DecodePgpKey("key with spaces")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "given PGP key cannot contain spaces")
}

func TestTestEncryptOkay(t *testing.T) {
	key := util.ReadKeyFile(t, publicFile)
	entity, err := DecodePgpKey(string(key))
	assert.Nil(t, err)
	err = TestEncrypt(entity)
	assert.Nil(t, err)
}

func TestTestEncryptNoEncryptionKey(t *testing.T) {
	key := util.ReadKeyFile(t, publicNoEncryptionFile)
	entity, err := DecodePgpKey(string(key))
	assert.Nil(t, err)
	err = TestEncrypt(entity)
	assert.NotNil(t, err)
	assert.Regexp(t, `error setting up encryption for PGP message: openpgp: invalid argument: cannot encrypt a message to key id .+ because it has no valid encryption keys`, err)
}

func TestTestEncryptFailExpired(t *testing.T) {
	key := util.ReadKeyFile(t, expiredFile)
	entity, err := DecodePgpKey(string(key))
	assert.NotNil(t, entity)
	assert.Nil(t, err)
	err = TestEncrypt(entity)
	assert.NotNil(t, err)
}

func TestDecodeAndArmorBase64Entity(t *testing.T) {
	cases := []struct {
		description    string
		input          string
		expectedError  error
		expectedOutput string
	}{
		{
			description:    "test DecodeAndArmorBase64Entity postitive case(key without checksum)",
			input:          string(util.ReadKeyFile(t, publicFile)),
			expectedOutput: string(util.ReadKeyFile(t, armoredKeyFile)),
		},
		{
			description:    "test DecodeAndArmorBase64Entity postitive case(key with checksum)",
			input:          string(util.ReadKeyFile(t, publicFile)) + "=FbQh\n", // added checksum
			expectedOutput: string(util.ReadKeyFile(t, armoredKeyFile)),
		},
		{
			description:   "test DecodeAndArmorBase64Entity postitive decode error",
			input:         "abc",
			expectedError: fmt.Errorf("%v", errors.New("error decoding given PGP key: illegal base64 data at input byte 0")),
		},
	}
	for _, testcase := range cases {
		output, err := DecodeAndArmorBase64Entity(testcase.input, constants.PublicKeyHeader)
		if err != nil {
			assert.Equal(t, testcase.expectedError.Error(), err.Error())
		} else {
			assert.Contains(t, output, testcase.input)
		}
	}
}
