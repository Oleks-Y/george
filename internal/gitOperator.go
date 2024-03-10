package internal

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

type GitOperator struct {
	path string
}

func NewGitOperator(path string) *GitOperator {
	cmd := exec.Command("git", "status")
	cmd.Dir = path

	err := cmd.Run()

	if err != nil {
		log.Fatalf("given path is not a git repository: %s", path)
	}

	return &GitOperator{
		path: path,
	}

}

func (g *GitOperator) FetchDiff() (string, error) {
	cmd := exec.Command("git", "--no-pager", "diff", "HEAD", "--no-commit-id")
	cmd.Dir = g.path

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}

	return out.String(), nil
}
