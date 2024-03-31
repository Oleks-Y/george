package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Oleks-Y/george/internal"
)

func main() {
	fmt.Println("Let's start!")

	gitPath := "./"

	if len(os.Args) > 1 {
		gitPath = os.Args[1]
	}

	responseChan := make(chan string, 1)

	confirmCh := make(chan bool, 1)

	var previewCh chan string
	var errCh chan error

	go func() {
		gitOp := internal.NewGitOperator(gitPath)
		openAI := internal.NewOpenAI()

		gitDiff, err := gitOp.FetchDiff()
		if err != nil {
			panic(err)
		}

		commitCandidates, err := openAI.GenCommits(*gitDiff)
		if err != nil {
			panic(err)
		}

		responseChan <- fmt.Sprintf("commit candidates: %v", commitCandidates)

		pCh, eCh, err := gitOp.MakeCommits(gitDiff, commitCandidates, confirmCh)
		if err != nil {
			panic(err)
		}

		previewCh = pCh
		errCh = eCh

	}()

	for {
		select {
		case previewFile := <-previewCh:
			confirm, err := printDiff(previewFile)
			if err != nil {
				fmt.Println("failed to print diff: ", err)
			}

			confirmCh <- confirm
		case response := <-responseChan:
			fmt.Println("Received response from ChatGPT: ")
			fmt.Println(response)
		case err := <-errCh:
			fmt.Println("failed to make commit: ", err)
		}
	}
}

func printDiff(diff string) (bool, error) {
	fmt.Println("\n\nCommit preview: ")
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+"):
			fmt.Printf("\033[32m%s\033[0m\n", line) // Green for added lines
		case strings.HasPrefix(line, "-"):
			fmt.Printf("\033[31m%s\033[0m\n", line) // Red for removed lines
		case strings.HasPrefix(line, "commit message"):
			fmt.Printf("\033[1m%s\033[0m\n", line) // Bold for commit message
		default:
			fmt.Println(line)
		}
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\nSelect an option:")
		fmt.Println("1. Accept")
		fmt.Println("2. Decline")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			return true, nil
		case "2":
			return false, nil
		default:
			fmt.Println("Invalid option, please try again.")
		}
	}
}
