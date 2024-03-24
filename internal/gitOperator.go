package internal

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type GitOperator struct {
	path string
}

type GitDiff struct {
	Files []FileDiff
}

type FileDiff struct {
	Header string
	Path   string
	Hunks  []Hunk
}

type Hunk struct {
	Id      int
	Content string
}

type FilePatch struct {
	FilePath string
	HunkIds  []int
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

func (g *GitOperator) FetchDiff() (*GitDiff, error) {
	cmd := exec.Command("git", "--no-pager", "diff", "HEAD", "--no-commit-id", "-U0")
	cmd.Dir = g.path

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get git diff: %w", err)
	}

	diff, err := parseDiff(out.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	return diff, nil
}

func (g *GitOperator) ApplyDiff() error {

	return nil
}

func parseDiff(diff string) (*GitDiff, error) {
	files := strings.SplitAfter(diff, "diff --git")
	fileDiffs := []FileDiff{}

	nextId := 0

	for _, file := range files {
		if file == "" || file == "diff --git" {
			continue
		}
		lines := strings.SplitAfter(file, "\n")
		hunks := []Hunk{}
		currentHunk := Hunk{}
		fileDiff := FileDiff{}

		for i, line := range lines {
			if i == 0 {
				fileDiff.Header += "diff --git " + line
				fileDiff.Path = strings.Trim(strings.Trim(line, " "), "\n")
			} else if i > 0 && i < 4 {
				fileDiff.Header += line
			} else {
				if strings.HasPrefix(line, "@@") {
					if currentHunk.Content != "" {
						hunks = append(hunks, currentHunk)
					}

					currentHunk = Hunk{
						Id:      nextId,
						Content: line,
					}

					nextId++
				} else {
					currentHunk.Content += line
				}
			}

		}

		fileDiff.Hunks = hunks
		fileDiffs = append(fileDiffs, fileDiff)
	}

	return &GitDiff{
		Files: fileDiffs,
	}, nil
}

func createPatch(diff *GitDiff, patchR []FilePatch) (string, error) {
	patch := ""

	for _, filePatch := range patchR {
		var fileDiff *FileDiff

		for _, file := range diff.Files {
			if file.Path == filePatch.FilePath {
				fileDiff = &file
				break
			}
		}

		if fileDiff == nil {
			return "", fmt.Errorf("file not found: %s", filePatch.FilePath)
		}

		hunks := []Hunk{}

		for _, hunk := range fileDiff.Hunks {
			for _, id := range filePatch.HunkIds {
				if hunk.Id == id {
					hunks = append(hunks, hunk)
					continue
				}
			}
		}

		if len(hunks) != len(filePatch.HunkIds) {
			return "", fmt.Errorf("hunk not found: %v", filePatch.HunkIds)
		}

		patch += fileDiff.Header

		for _, hunk := range hunks {
			patch += hunk.Content
		}
	}

	return patch, nil
}
