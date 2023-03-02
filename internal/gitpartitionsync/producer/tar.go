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

const TAR_DIRECTORY = "tars"

func (g *GitPartitionSyncProducer) tarRepos(repoPath string, sync syncConfig) (string, error) {
	err := g.clean(TAR_DIRECTORY)
	if err != nil {
		return "", err
	}

	// ensure the repo actually exists before trying to tar it
	if _, err := os.Stat(repoPath); err != nil {
		return "", fmt.Errorf("Unable to tar files - %v", err.Error())
	}

	tarPath := fmt.Sprintf("%s/%s/%s.tar", g.config.Workdir, TAR_DIRECTORY, sync.SourceProjectName)
	f, err := os.Create(tarPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// credit: https://medium.com/@skdomino/taring-untaring-files-in-go-6b07cf56bc07
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
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		// Closing a healthy file twice yields a syscall.EINVAL
		// error which is safe to discard in this case.
		defer func() { _ = f.Close() }()

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; defering would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})

	if err != nil {
		return "", err
	}

	return tarPath, nil
}
