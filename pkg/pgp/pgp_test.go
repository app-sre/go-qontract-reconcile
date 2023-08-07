package pgp

import (
	"testing"

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
)

func TestDecodePgpKeyFailDecode(t *testing.T) {
	entity, err := DecodePgpKey("a", "/test/path/file.yml")
	assert.Nil(t, entity)
	assert.EqualError(t, err, "error decoding given PGP key: illegal base64 data at input byte 0")
}

func TestDecodePgpKeyFailEntity(t *testing.T) {
	entity, err := DecodePgpKey("Zm9vCg==", "/test/path/file.yml")
	assert.Nil(t, entity)
	assert.EqualError(t, err, "error parsing given PGP key: openpgp: invalid data: tag byte does not have MSB set")
}

func TestDecodePgpKeyOkay(t *testing.T) {
	key := util.ReadKeyFile(t, publicFile)
	entity, err := DecodePgpKey(string(key), "/test/path/file.yml")
	assert.Nil(t, err)
	assert.NotNil(t, entity)

	key = util.ReadKeyFile(t, publicSingleLineFile)
	entity, err = DecodePgpKey(string(key), "/test/path/file.yml")
	assert.Nil(t, err)
	assert.NotNil(t, entity)
}

func TestTestDecodeEccOkay(t *testing.T) {
	key := util.ReadKeyFile(t, eccFile)
	entity, err := DecodePgpKey(string(key), "/test/path/file.yml")
	assert.Nil(t, err)
	assert.NotNil(t, entity)
}

func TestDecodePgpKeyInvalidArmoredKey(t *testing.T) {
	_, err := DecodePgpKey("-----BEGIN PGP PUBLIC KEY BLOCK-----", "/test/path/file.yml")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "ASCII-armored PGP keys are not supported; please remove type headers and checksum")

	key := util.ReadKeyFile(t, publicFile)
	// The CRC24 checksum for this public key is: 15b421 (encoded as =FbQh),
	// add valid CRC at the end of the key to simulate a common mistake.
	_, err = DecodePgpKey(string(key)+"\n=FbQh\n", "/test/path/file.yml")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "ASCII-armored PGP keys are not supported; please remove checksum (encoded as =FbQh)")
}

func TestDecodePgpKeyInvalidArmoredKeyChecksum(t *testing.T) {
	key := util.ReadKeyFile(t, publicFile)
	// The CRC24 checksum for this public key is: 15b421 (encoded as =FbQh).
	// add an invalid CRC at the end of the key.
	_, err := DecodePgpKey(string(key)+"\n=T3st\n", "/test/path/file.yml")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "error decoding given ASCII-armored PGP key: openpgp: invalid data: armor invalid")
}

func TestDecodePgpKeyInvalidPrivateKey(t *testing.T) {
	key := util.ReadKeyFile(t, privateFile)
	_, err := DecodePgpKey(string(key), "/test/path/file.yml")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "given PGP key is not a Public Key")
}

func TestDecodePgpKeyInvalidSpaces(t *testing.T) {
	_, err := DecodePgpKey("key with spaces", "/test/path/file.yml")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "given PGP key cannot contain spaces")
}

func TestTestEncryptOkay(t *testing.T) {
	key := util.ReadKeyFile(t, publicFile)
	entity, err := DecodePgpKey(string(key), "/test/path/file.yml")
	assert.Nil(t, err)
	err = TestEncrypt(entity)
	assert.Nil(t, err)
}

func TestTestEncryptNoEncryptionKey(t *testing.T) {
	key := util.ReadKeyFile(t, publicNoEncryptionFile)
	entity, err := DecodePgpKey(string(key), "/test/path/file.yml")
	assert.Nil(t, err)
	err = TestEncrypt(entity)
	assert.NotNil(t, err)
	assert.Regexp(t, `error setting up encryption for PGP message: openpgp: invalid argument: cannot encrypt a message to key id .+ because it has no valid encryption keys`, err)
}

func TestTestEncryptFailExpired(t *testing.T) {
	key := util.ReadKeyFile(t, expiredFile)
	entity, err := DecodePgpKey(string(key), "/test/path/file.yml")
	assert.NotNil(t, entity)
	assert.Nil(t, err)
	err = TestEncrypt(entity)
	assert.NotNil(t, err)
}

func TestCRC24(t *testing.T) {
	cases := []struct {
		description string
		given       []byte
		expected    uint32
	}{
		{"EmptyByteArray", []byte{}, 0xb704ce}, // This is the same as the used polynominal.
		{"WithOneZeroByte", []byte{0}, 0x6169d3},
		{"WithTwoZeroBytes", []byte{0, 0}, 0xfaedc0},
		{"WithFourZeroBytes", []byte{0, 0, 0, 0}, 0xf659f3},
		{"WithFourBytes", []byte{1, 2, 3, 4}, 0x7878cd},
		{"WithSimpleString", []byte("test"), 0xf86ed0},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, crc24(tc.given))
		})
	}
}
