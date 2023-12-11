// Package pgp provides functions to work with PGP keys.
package pgp

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	pgperr "github.com/ProtonMail/go-crypto/openpgp/errors"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	parmor "github.com/ProtonMail/gopenpgp/v2/armor"
)

// TestEncrypt tests if an opengpg.Entity can be used for encryption
func TestEncrypt(entity *openpgp.Entity) error {
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

func keyArmor(anchor string) string {
	return fmt.Sprintf("-----%s %s-----", anchor, openpgp.PublicKeyType)
}

// DecodePgpKey tests if the passed in pgpKey is a base64 encoded pgp Public Key.
func DecodePgpKey(pgpKey string) (*openpgp.Entity, error) {
	pgpKey = strings.TrimRight(pgpKey, " \n\r")
	pgpKey = strings.TrimSpace(pgpKey)

	keyArmorStart := keyArmor("BEGIN")

	if strings.HasPrefix(pgpKey, keyArmorStart[:strings.Index(keyArmorStart, " ")]) {
		return nil, errors.New("please remove type headers")
	}

	if strings.Contains(pgpKey, " ") {
		return nil, fmt.Errorf("given PGP key cannot contain spaces")
	}

	// decodeAndValidatePGPkey supports ASCII armored gpg keys and return key in case key+checksum provided
	data, err := decodePGPkey(pgpKey)
	if err != nil {
		return nil, err
	}

	packets := packet.NewReader(bytes.NewBuffer(data))

	p, err := packets.Next()
	if err != nil {
		return nil, fmt.Errorf("error parsing given PGP key: %w", err)
	}
	if _, ok := p.(*packet.PublicKey); !ok {
		return nil, fmt.Errorf("given PGP key is not a Public Key")
	}
	packets.Unread(p)

	entity, err := openpgp.ReadEntity(packets)
	if err != nil {
		return nil, fmt.Errorf("error parsing given PGP key: %w", err)
	}

	return entity, nil
}

// DecodeAndArmorBase64Entity decodes a base64 encoded entity and armors it.
func DecodeAndArmorBase64Entity(encodedEntity string, armorType string) (string, error) {
	decodedEntity, err := decodePGPkey(encodedEntity)
	if err != nil {
		return "", err
	}
	return parmor.ArmorWithType([]byte(decodedEntity), armorType)
}

// decodePGPkey PGP key is a wrapper function
// around golang standard base64 package
// which will decode keys with checksum as well as non checksum and return  bytes from key part
func decodePGPkey(encodedKeyData string) ([]byte, error) {
	// check if the string is standard PGP key with base64 encoding WITHOUT checksum
	decodedKeyData, err := base64.StdEncoding.DecodeString(encodedKeyData)
	if err != nil {
		// check if keys is encoded using OpenPGP's Radix-64 encoding
		// save the actual error
		decodeErr := err
		// create ascii armored gpg key string by adding
		// ------ BEGIN -------
		// encodedKeyData (base64key + '=' + checksum)
		// ------  END  -------

		// remove trailing newLines if any
		encodedKeyData = strings.Trim(encodedKeyData, "\n")
		pgpKey := fmt.Sprintf("%s\n\n%s\n%s", keyArmor("BEGIN"), encodedKeyData, keyArmor("END"))
		// unarmor the encodedKeyData to get the encoded pgp key part also checking if added checksum is valid
		encodedBytes, err := parmor.Unarmor(pgpKey)
		if err != nil {
			if _, ok := err.(pgperr.StructuralError); ok {
				return nil, fmt.Errorf("error decoding given ASCII-armored PGP key: %w", err)
			}
			return nil, fmt.Errorf("error decoding given PGP key: %w", decodeErr)
		}
		return encodedBytes, nil
	}
	return decodedKeyData, nil
}
