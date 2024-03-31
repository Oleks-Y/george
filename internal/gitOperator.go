package internal

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FileCommitPreview struct {
	CommitMsg string
	Path      string
}

type GitOperator struct {
	path       string
	patchFiles []FileCommitPreview
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

type FilePatchRequest struct {
	FilePath string `json:"filePath"`
	HunkIds  []int  `json:"hunkIds"`
}

func NewGitOperator(path string) *GitOperator {
	cmd := exec.Command("git", "status")
	cmd.Dir = path
	appDir := filepath.Join(path, ".george")
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		err := os.Mkdir(appDir, 0755)
		if err != nil {
			log.Fatalf("failed to create .george directory: %s", err)
		}
	}

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

func (g *GitOperator) PreviewPatch(gitDiff *GitDiff, commit CommitCandidate) (string, string, error) {
	appDir := filepath.Join(g.path, ".george")

	patch, err := createPatch(gitDiff, commit.Files)
	if err != nil {
		return "", "", err
	}

	tempFile, err := os.CreateTemp(appDir, "patch-*")
	if err != nil {
		return "", "", err
	}

	patchPreview := fmt.Sprintf("commit message: %s\n\n%s", commit.CommitMessage, patch)
	if _, err := tempFile.WriteString(patch); err != nil {
		return "", "", err
	}

	if err := tempFile.Close(); err != nil {
		return "", "", err
	}

	aboslutePath, err := filepath.Abs(tempFile.Name())
	if err != nil {
		return "", "", err
	}

	g.patchFiles = append(g.patchFiles, FileCommitPreview{
		CommitMsg: commit.CommitMessage,
		Path:      aboslutePath,
	})

	return patchPreview, aboslutePath, nil
}

func (g *GitOperator) ApplyPatch(patchPath string) error {
	var preview FileCommitPreview
	for _, p := range g.patchFiles {
		if p.Path == patchPath {
			preview = p
			break
		}
	}

	cmd := exec.Command("git", "apply", "--index", preview.Path)
	cmd.Dir = g.path

	fmt.Println(cmd.String())

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to apply patch: %w", err)
	}

	cmd = exec.Command("git", "commit", "-m", preview.CommitMsg)
	cmd.Dir = g.path

	fmt.Println(cmd.String())

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

func (g *GitOperator) MakeCommits(gitDiff *GitDiff, commits []CommitCandidate, confirmationCh chan bool) (chan string, chan error, error) {
	previewCh := make(chan string, len(commits))
	errCh := make(chan error, len(commits))

	go func() {
		for _, commit := range commits {
			preview, patchFile, err := g.PreviewPatch(gitDiff, commit)
			if err != nil {
				errCh <- fmt.Errorf("failed to preview patch: %w", err)
				return
			}

			previewCh <- preview

			confirm := <-confirmationCh

			if !confirm {
				continue
			}

			fmt.Println("committing...")
			err = g.ApplyPatch(patchFile)

			if err != nil {
				panic(err)
				// errCh <- fmt.Errorf("failed to apply patch: %w", err)
				// return
			}
		}
	}()

	return previewCh, errCh, nil
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

func createPatch(diff *GitDiff, patchR []FilePatchRequest) (string, error) {
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
