package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Oleks-Y/george/internal"
)

func main() {
	fmt.Println("Let's start!")

	gitPath := "./"

	if len(os.Args) > 1 {
		gitPath = os.Args[1]
	}

	fmt.Println("using git path: ", gitPath)

	gitOp := internal.NewGitOperator(gitPath)
	gpt := internal.NewOpenAI()

	gitDiff, err := gitOp.FetchDiff()

	if err != nil {
		log.Fatal("failed to get git diff: ", err)
	}

	commits, err := gpt.GenCommits(gitDiff)
	if err != nil {
		log.Fatal("failed to generate commits: ", err)
	}

	fmt.Println(commits)
}
