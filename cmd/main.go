package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/Oleks-Y/george/internal"
)

// func main1() {
// 	fmt.Println("Let's start!")

// 	gitPath := "./"

// 	app := tview.NewApplication()

// 	if len(os.Args) > 1 {
// 		gitPath = os.Args[1]
// 	}

// 	fmt.Println("using git path: ", gitPath)

// 	gitOp := internal.NewGitOperator(gitPath)
// 	openAI := internal.NewOpenAI()

// 	gitDiff, err := gitOp.FetchDiff()

// 	if err != nil {
// 		fmt.Println("failed to get git diff: ", err)
// 	}

// 	commitCandidates, err := openAI.GenCommits(*gitDiff)
// 	if err != nil {
// 		fmt.Println("failed to generate commit candidates: ", err)
// 	}

// 	fmt.Println("commit candidates: ", commitCandidates)

// //TODO: send openAI request
// patchRequest := internal.FilePatchRequest{
// 	FilePath: "a/src/components/InputField/InputField.tsx b/src/components/InputField/InputField.tsx",
// 	HunkIds:  []int{11},
// }

// patchPreviewChan := make(chan string, 1)
// errChan := make(chan error, 1)

// go func() {
// 	patchP, err := gitOp.PreviewPatch(gitDiff, []internal.FilePatchRequest{patchRequest})
// 	if err != nil {
// 		errChan <- err
// 	}
// 	patchPreviewChan <- patchP
// }()

// //TODO: save patch to file and open in editor to leave user with save/reject options

// select {
// case patchPreview := <-patchPreviewChan:
// 	list := tview.NewList().
// 		AddItem("Accept", "Accept the patch", 'a', func() {
// 			// Handle acceptance of the patch
// 			app.Stop()
// 		}).
// 		AddItem("Abort", "Abort the patch", 'b', func() {
// 			// Handle aborting of the patch
// 			app.Stop()
// 		})

// 	textView := tview.NewTextView().
// 		SetDynamicColors(true).
// 		SetRegions(true).
// 		SetChangedFunc(func() {
// 			app.Draw()
// 		})

// 	textView.SetBorder(true).SetTitle("Patch Preview")

// 	fmt.Fprint(textView, patchPreview)

// 	flex := tview.NewFlex().
// 		SetDirection(tview.FlexRow).
// 		AddItem(textView, 0, 1, false).
// 		AddItem(list, 0, 1, true)

// 	if err := app.SetRoot(flex, true).Run(); err != nil {
// 		panic(err)
// 	}
// case err := <-errChan:
// 	fmt.Println("failed to preview patch: ", err)
// }
// }

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
			log.Fatalf("failed to get git diff: %v", err)
			return
		}

		commitCandidates, err := openAI.GenCommits(*gitDiff)
		if err != nil {
			log.Fatalf("failed to get git diff: %v", err)
			return
		}

		responseChan <- fmt.Sprintf("commit candidates: %v", commitCandidates)

		pCh, eCh, err := gitOp.MakeCommits(gitDiff, commitCandidates, confirmCh)
		if err != nil {
			log.Fatalf("failed to make commits: %v", err)
			return
		}

		previewCh = pCh
		errCh = eCh

	}()

	resp := <-responseChan
	fmt.Println(resp)

	for {
		select {
		case previewFile := <-previewCh:
			fmt.Println(previewFile)
			err := exec.Command("code", previewFile).Run()

			if err != nil {
				fmt.Println("failed to open file in editor: ", err)
			}

			confirmCh <- false
		case err := <-errCh:
			fmt.Println("failed to make commit: ", err)
		}
	}
}
