package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/app-sre/user-validator/internal/queries"
	. "github.com/app-sre/user-validator/pkg"
	"github.com/google/go-github/v42/github"
	"github.com/stretchr/testify/assert"
)

var (
	publicFile             = "../test/data/public_key.b64"
	publicSingleLineFile   = "../test/data/public_key.single-line.b64"
	publicNoEncryptionFile = "../test/data/public_key.no-encryption.b64"
	privateFile            = "../test/data/private_key.b64"
	expiredFile            = "../test/data/expired_key.b64"
	eccFile                = "../test/data/ecc_key.b64"
)

func readKeyFile(t *testing.T, fileName string) []byte {
	key, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("Could not read public key test data %s, error: %s", fileName, err.Error())
	}
	return key
}

func TestDecodePgpKeyFailDecode(t *testing.T) {
	entity, err := decodePgpKey("a", "/test/path/file.yml")
	assert.Nil(t, entity)
	assert.EqualError(t, err, "error decoding given PGP key: illegal base64 data at input byte 0")
}

func TestDecodePgpKeyFailEntity(t *testing.T) {
	entity, err := decodePgpKey("Zm9vCg==", "/test/path/file.yml")
	assert.Nil(t, entity)
	assert.EqualError(t, err, "error parsing given PGP key: openpgp: invalid data: tag byte does not have MSB set")
}

func TestDecodePgpKeyOkay(t *testing.T) {
	key := readKeyFile(t, publicFile)
	entity, err := decodePgpKey(string(key), "/test/path/file.yml")
	assert.Nil(t, err)
	assert.NotNil(t, entity)

	key = readKeyFile(t, publicSingleLineFile)
	entity, err = decodePgpKey(string(key), "/test/path/file.yml")
	assert.Nil(t, err)
	assert.NotNil(t, entity)
}

func TestTestDecodeEccFailed(t *testing.T) {
	key := readKeyFile(t, eccFile)
	_, err := decodePgpKey(string(key), "/test/path/file.yml")
	assert.NotNil(t, err)
}

func TestDecodePgpKeyInvalidArmoredKey(t *testing.T) {
	_, err := decodePgpKey("-----BEGIN PGP PUBLIC KEY BLOCK-----", "/test/path/file.yml")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "ASCII-armored PGP keys are not supported; please remove type headers and checksum")

	key := readKeyFile(t, publicFile)
	// The CRC24 checksum for this public key is: 15b421 (encoded as =FbQh),
	// add valid CRC at the end of the key to simulate a common mistake.
	_, err = decodePgpKey(string(key)+"\n=FbQh\n", "/test/path/file.yml")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "ASCII-armored PGP keys are not supported; please remove checksum (encoded as =FbQh)")
}

func TestDecodePgpKeyInvalidArmoredKeyChecksum(t *testing.T) {
	key := readKeyFile(t, publicFile)
	// The CRC24 checksum for this public key is: 15b421 (encoded as =FbQh).
	// add an invalid CRC at the end of the key.
	_, err := decodePgpKey(string(key)+"\n=T3st\n", "/test/path/file.yml")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "error decoding given ASCII-armored PGP key: openpgp: invalid data: armor invalid")
}

func TestDecodePgpKeyInvalidPrivateKey(t *testing.T) {
	key := readKeyFile(t, privateFile)
	_, err := decodePgpKey(string(key), "/test/path/file.yml")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "given PGP key is not a Public Key")
}

func TestDecodePgpKeyInvalidSpaces(t *testing.T) {
	_, err := decodePgpKey("key with spaces", "/test/path/file.yml")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "given PGP key cannot contain spaces")
}

func TestTestEncryptOkay(t *testing.T) {
	key := readKeyFile(t, publicFile)
	entity, err := decodePgpKey(string(key), "/test/path/file.yml")
	assert.Nil(t, err)
	err = testEncrypt(entity)
	assert.Nil(t, err)
}

func TestTestEncryptNoEncryptionKey(t *testing.T) {
	key := readKeyFile(t, publicNoEncryptionFile)
	entity, err := decodePgpKey(string(key), "/test/path/file.yml")
	assert.Nil(t, err)
	err = testEncrypt(entity)
	assert.NotNil(t, err)
	assert.Regexp(t, `cannot encrypt a message .+ no encryption keys`, err)
}

func TestTestEncryptFailExpired(t *testing.T) {
	key := readKeyFile(t, expiredFile)
	entity, err := decodePgpKey(string(key), "/test/path/file.yml")
	assert.NotNil(t, entity)
	assert.Nil(t, err)
	err = testEncrypt(entity)
	assert.NotNil(t, err)
}

func TestValidatePgpKeysValid(t *testing.T) {
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{}
	userResponse := queries.UsersResponse{
		Users_v1: []queries.UsersUsers_v1User_v1{{
			Public_gpg_key: string(readKeyFile(t, publicFile)),
		}},
	}
	validationErrors := v.validatePgpKeys(userResponse)
	assert.Len(t, validationErrors, 0)
}

func TestValidatePgpKeysInValid(t *testing.T) {
	// Todo add fixture for expired key
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{}
	userResponse := queries.UsersResponse{
		Users_v1: []queries.UsersUsers_v1User_v1{{
			Path:           "/foo/bar",
			Public_gpg_key: "a",
		}},
	}
	validationErrors := v.validatePgpKeys(userResponse)
	assert.Len(t, validationErrors, 1)
	assert.Equal(t, "validatePgpKeys", validationErrors[0].Validation)
	assert.Equal(t, "/foo/bar", validationErrors[0].Path)
	assert.EqualError(t, validationErrors[0].Error, "error decoding given PGP key: illegal base64 data at input byte 0")
}

func TestValidateValidateUsersSinglePathInValid(t *testing.T) {
	// Todo add fixture for expired key
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{}
	userResponse := queries.UsersResponse{
		Users_v1: []queries.UsersUsers_v1User_v1{{
			Path:         "/foo/bar",
			Org_username: "foo",
		}, {
			Path:         "/foo/rab",
			Org_username: "foo",
		}},
	}
	validationErrors := v.validateUsersSinglePath(userResponse)
	assert.Len(t, validationErrors, 2)
	assert.Equal(t, "validateUsersSinglePath", validationErrors[0].Validation)
	assert.Equal(t, "/foo/bar", validationErrors[0].Path)
	assert.Equal(t, "/foo/rab", validationErrors[1].Path)
	assert.EqualError(t, validationErrors[0].Error, "user \"foo\" has multiple user files")
}

func TestValidateValidateUsersSinglePathValid(t *testing.T) {
	// Todo add fixture for expired key
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{}
	userResponse := queries.UsersResponse{
		Users_v1: []queries.UsersUsers_v1User_v1{{
			Path:         "/foo/bar",
			Org_username: "foo",
		}, {
			Path:         "/foo/rab",
			Org_username: "rab",
		}},
	}
	validationErrors := v.validateUsersSinglePath(userResponse)
	assert.Len(t, validationErrors, 0)
}

func createGithubUsersMock(t *testing.T, retBody string, retCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/v3/users")
		_, err := io.ReadAll(r.Body)
		assert.Nil(t, err)

		fmt.Fprint(w, retBody)

	}))
}

func TestGetAndValidateUserOK(t *testing.T) {
	var err error
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{
		Concurrency: 1,
	}

	githubMock := createGithubUsersMock(t, `{"login": "bar"}`, 200)

	gh, err := github.NewEnterpriseClient(githubMock.URL, githubMock.URL, http.DefaultClient)
	assert.Nil(t, err)

	v.AuthenticatedGithubClient = &AuthenticatedGithubClient{
		GithubClient: gh,
	}

	validationError := v.getAndValidateUser(context.Background(), queries.UsersUsers_v1User_v1{
		Path:            "/foo/bar",
		Github_username: "bar",
	})
	assert.Nil(t, validationError)
}

func TestGetAndValidateUserFailed(t *testing.T) {
	var err error
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{
		Concurrency: 1,
	}

	githubMock := createGithubUsersMock(t, `{"login": "bar"}`, 200)

	gh, err := github.NewEnterpriseClient(githubMock.URL, githubMock.URL, http.DefaultClient)
	assert.Nil(t, err)

	v.AuthenticatedGithubClient = &AuthenticatedGithubClient{
		GithubClient: gh,
	}

	validationError := v.getAndValidateUser(context.Background(), queries.UsersUsers_v1User_v1{
		Path:            "/foo/bar",
		Github_username: "Bar",
	})
	assert.NotNil(t, validationError)
	assert.Equal(t, "validateUsersGithub", validationError.Validation)
	assert.Equal(t, "/foo/bar", validationError.Path)
	assert.EqualError(t, validationError.Error, "Github username is case sensitive in OSD. GithubUsername \"bar\", configured Username \"Bar\"")
}

func TestGetAndValidateUserApiFailed(t *testing.T) {
	var err error
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{
		Concurrency: 1,
	}
	githubMock := createGithubUsersMock(t, `{}`, 500)

	gh, err := github.NewEnterpriseClient(githubMock.URL, githubMock.URL, http.DefaultClient)
	assert.Nil(t, err)

	v.AuthenticatedGithubClient = &AuthenticatedGithubClient{
		GithubClient: gh,
	}

	validationError := v.getAndValidateUser(context.Background(), queries.UsersUsers_v1User_v1{
		Path:            "/foo/bar",
		Github_username: "bar",
	})
	assert.NotNil(t, validationError)
	assert.Equal(t, "validateUsersGithub", validationError.Validation)
	assert.Equal(t, "/foo/bar", validationError.Path)
	// Just assert an error, it could vary ...
	assert.Error(t, validationError.Error)
}

func TestValidateUsersGithubErrorsReturned(t *testing.T) {
	var err error
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{
		Concurrency: 1,
	}

	v.githubValidateFunc = v.getAndValidateUser

	githubMock := createGithubUsersMock(t, `{"login": "bar"}`, 200)

	gh, err := github.NewEnterpriseClient(githubMock.URL, githubMock.URL, http.DefaultClient)
	assert.Nil(t, err)

	v.AuthenticatedGithubClient = &AuthenticatedGithubClient{
		GithubClient: gh,
	}

	validationErrors := v.validateUsersGithub(context.Background(), queries.UsersResponse{
		Users_v1: []queries.UsersUsers_v1User_v1{{
			Path:            "/foo/bar",
			Github_username: "Bar",
		}},
	})
	assert.NotNil(t, validationErrors)
	assert.Len(t, validationErrors, 1)
}

func TestValidateUsersGithubCallingValidate(t *testing.T) {
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{
		Concurrency: 1,
	}
	validated := false
	v.githubValidateFunc = func(ctx context.Context, user queries.UsersUsers_v1User_v1) *ValidationError {
		validated = true
		return nil
	}

	v.validateUsersGithub(context.Background(), queries.UsersResponse{
		Users_v1: []queries.UsersUsers_v1User_v1{{
			Path:            "/foo/bar",
			Github_username: "bar",
		}},
	})
	assert.True(t, validated)
}

func TestValidateUsersGithubValidateError(t *testing.T) {
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{
		Concurrency: 1,
	}
	v.githubValidateFunc = func(ctx context.Context, user queries.UsersUsers_v1User_v1) *ValidationError {
		return &ValidationError{}
	}

	v.validateUsersGithub(context.Background(), queries.UsersResponse{
		Users_v1: []queries.UsersUsers_v1User_v1{{
			Path:            "/foo/bar",
			Github_username: "bar",
		}, {
			Path:            "/foo/bar",
			Github_username: "bar",
		}},
	})
}

func TestRemoveInvalidUsers(t *testing.T) {
	v := ValidateUser{}
	v.ValidateUserConfig = &ValidateUserConfig{
		Concurrency:  1,
		InvalidUsers: "/foo/bar",
	}

	users := queries.UsersResponse{
		Users_v1: []queries.UsersUsers_v1User_v1{{
			Path: "/foo/bar",
		}, {
			Path: "/bar/foo",
		},
		},
	}

	validUser := v.removeInvalidUsers(&users)
	assert.Len(t, validUser.GetUsers_v1(), 1)
	assert.Equal(t, validUser.GetUsers_v1()[0].Path, "/bar/foo")
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
