package producer

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

const CLONE_DIRECTORY = "glrepos"

func (g *GitPartitionSyncProducer) cloneRepos(sync syncConfig) (string, error) {
	err := g.clean(CLONE_DIRECTORY)
	if err != nil {
		return "", err
	}

	authURL, err := g.formatAuthURL(fmt.Sprintf("%s/%s", sync.SourceProjectGroup, sync.SourceProjectName))
	if err != nil {
		return "", err
	}

	args := []string{"-c", fmt.Sprintf("git clone %s", authURL)}
	cmd := exec.Command("/bin/sh", args...)
	cmd.Dir = fmt.Sprintf("%s/%s", g.config.Workdir, CLONE_DIRECTORY)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return "", errors.New(strings.ReplaceAll(stderr.String(), g.config.GlToken, "[REDACTED]"))
	}

	return fmt.Sprintf("%s/%s/%s", g.config.Workdir, CLONE_DIRECTORY, sync.SourceProjectName), nil
}

// returns git user-auth format of remote url
func (g *GitPartitionSyncProducer) formatAuthURL(pid string) (string, error) {
	projectURL := fmt.Sprintf("%s/%s", g.config.GlBaseURL, pid)
	parsedURL, err := url.Parse(projectURL)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s:%s@%s%s.git",
		parsedURL.Scheme,
		g.config.GlUsername,
		g.config.GlToken,
		parsedURL.Host,
		parsedURL.Path,
	), nil
}
