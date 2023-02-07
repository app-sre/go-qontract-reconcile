package producer

import (
	"fmt"
	"log"
	"os"

	"filippo.io/age"
)

const ENCRYPT_DIRECTORY = "encrypted"

// utilizes x25519 to output encrypted tars
func (g *GitPartitionSyncProducer) encryptRepoTars(tarPath string, sync syncConfig) (string, error) {
	err := g.clean(ENCRYPT_DIRECTORY)
	if err != nil {
		return "", err
	}

	recipient, err := age.ParseX25519Recipient(g.config.PublicKey)
	if err != nil {
		log.Fatalf("Failed to parse public key %q: %v", g.config.PublicKey, err)
	}

	encryptPath := fmt.Sprintf("%s/%s/%s.tar.age", g.config.Workdir, ENCRYPT_DIRECTORY, sync.SourceProjectName)
	f, err := os.Create(encryptPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// read in tar data
	tarBytes, err := os.ReadFile(tarPath)
	if err != nil {
		return "", err
	}

	// encrypt
	encWriter, err := age.Encrypt(f, recipient)
	if err != nil {
		return "", err
	}
	encWriter.Write(tarBytes)

	if err := encWriter.Close(); err != nil {
		return "", err
	}

	return encryptPath, nil
}
