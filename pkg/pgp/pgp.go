package pgp

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	pgperr "github.com/ProtonMail/go-crypto/openpgp/errors"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	parmor "github.com/ProtonMail/gopenpgp/v2/armor"
	"github.com/app-sre/go-qontract-reconcile/pkg/util"
)

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

// crc24 calculates the CRC24 checksum OpenPGP variant for a given byte array.
//
// See the RFC 4880, "OpenPGP Message Format", Section 6.1, for source of this implementation.
func crc24(bytes []byte) uint32 {
	const (
		seed = 0xb704ce
		poly = 0x1864cfb
		mask = 0xffffff
	)

	var crc uint32 = seed

	for _, b := range bytes {
		crc ^= uint32(b) << 16
		for i := 0; i < 8; i++ {
			crc <<= 1
			if crc&0x1000000 != 0 {
				crc ^= poly
			}
		}
	}

	return crc & mask
}

func DecodePgpKey(pgpKey, path string) (*openpgp.Entity, error) {
	pgpKey = strings.TrimRight(pgpKey, " \n\r")
	pgpKey = strings.TrimSpace(pgpKey)

	keyArmor := func(anchor string) string {
		return fmt.Sprintf("-----%s %s-----", anchor, openpgp.PublicKeyType)
	}
	keyArmorStart := keyArmor("BEGIN")

	if strings.HasPrefix(pgpKey, keyArmorStart[:strings.Index(keyArmorStart, " ")]) {
		return nil, errors.New("ASCII-armored PGP keys are not supported; please remove type headers and checksum")
	}

	if strings.Contains(pgpKey, " ") {
		return nil, fmt.Errorf("given PGP key cannot contain spaces")
	}

	data, err := base64.StdEncoding.DecodeString(pgpKey)
	if err != nil {
		// Save the original Base64 decoder error,
		// to return if an error is not related to
		// ASCII armor parsing or validation.
		decodeErr := err

		pgpKey = fmt.Sprintf("%s\n\n%s\n%s", keyArmorStart, pgpKey, keyArmor("END"))
		block, err := armor.Decode(strings.NewReader(pgpKey))
		if err != nil {
			return nil, fmt.Errorf("error decoding given ASCII-armored PGP key: %w", err)
		}

		var body bytes.Buffer

		// Drain the Reader buffer, which causes the CRC24
		// checksum to be computed for the given ASCII armor.
		_, err = io.Copy(&body, block.Body)
		if err != nil {
			if _, ok := err.(pgperr.StructuralError); ok {
				return nil, fmt.Errorf("error decoding given ASCII-armored PGP key: %w", err)
			}
			return nil, fmt.Errorf("error decoding given PGP key: %w", decodeErr)
		}
		crc := crc24(body.Bytes())

		var crcBytes = []byte{0, 0, 0, 0}
		base64.StdEncoding.Encode(crcBytes, []byte{byte(crc >> 16), byte(crc >> 8), byte(crc)})
		crcBytesEncoded := fmt.Sprintf("=%s", string(crcBytes))

		util.Log().Debugw("A valid ASCII-armored PGP key has been given",
			"crc24_checksum", fmt.Sprintf("%x", crc),
			"crc24_encoded", crcBytesEncoded,
			"path", path,
		)

		return nil, fmt.Errorf("ASCII-armored PGP keys are not supported; please remove checksum (encoded as %s)", crcBytesEncoded)
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

func DecodeAndArmorBase64Entity(encodedEntity string, armorType string) (string, error) {
	decodedEntity, err := base64.StdEncoding.DecodeString(encodedEntity)
	if err != nil {
		return "", err
	}
	return parmor.ArmorWithType([]byte(decodedEntity), armorType)
}
