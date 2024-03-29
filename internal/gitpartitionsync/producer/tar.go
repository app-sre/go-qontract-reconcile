package producer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const tarDirectory = "tars"

func (g *GitPartitionSyncProducer) tarRepos(repoPath string, sync syncConfig) (string, error) {
	err := g.clean(tarDirectory)
	if err != nil {
		return "", err
	}

	// ensure the repo actually exists before trying to tar it
	if _, err := os.Stat(repoPath); err != nil {
		return "", fmt.Errorf("Unable to tar files - %v", err.Error())
	}

	tarPath := fmt.Sprintf("%s/%s/%s.tar", g.config.Workdir, tarDirectory, sync.SourceProjectName)
	f, err := os.Create(filepath.Clean(tarPath))
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	err = filepath.Walk(repoPath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, repoPath, "", -1), string(filepath.Separator))

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// open files for taring
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			return err
		}
		// Closing a healthy file twice yields a syscall.EINVAL
		// error which is safe to discard in this case.
		defer f.Close()

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return tarPath, nil
}
