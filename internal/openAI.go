package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type OpenAI struct {
	url          string
	apiKey       string
	conversation []Message
}

type ResponseBody struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int       `json:"created"`
	Model   string    `json:"model"`
	Usage   Usage     `json:"usage"`
	Choices []Choices `json:"choices"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Choices struct {
	FinishReason string  `json:"finish_reason"`
	Message      Message `json:"message"`
}

type CommitCandidate struct {
	CommitMessage string       `json:"commitMessage"`
	Files         []CommitFile `json:"files"`
}

type CommitFile struct {
	Path           string `json:"path"`
	LinesForCommit []int  `json:"linesForCommit"`
}

func NewOpenAI() *OpenAI {
	apiKey := os.Getenv("GEORGE_OPENAI_API_KEY")

	if apiKey == "" {
		log.Fatal("GEORGE_OPENAI_API_KEY is not set. Please set it and try again.")
	}

	defaultMsgContent := `
		You're CLI assistant for a software project.
		You're purpose is to group files for git commit messages in a meaningful way.
		Files should be grouped based on logical changes 
		and have a commit message that reflects the changes using Conventional Commits.
		As a response, provide a JSON array of objects with the following structure, excluding spaces and new lines:

		[
			{
				"commitMessage": "feat: example commit message",
				"files" : [
					{
						"path": "path/to/file",
						"linesForCommit": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
					}
				]
			}, 
			{
				"commitMessage": "fix: example commit message",
				"files" : [
					{
						"path": "path/to/file",
						"linesForCommit": [10, 11]
					}
				]
			}
		]
		As an input you will receive a git diff output.
		Create as many commits, as it is necessary to group the changes in a meaningful way 
		and each commit to be minimalistic.
		Commit can include more than one file grouped by same type of change, same context or when changes are related. 
		For changes in files, which are not related, create separate commits.
		For updates of dependencies, commit config files together with changes of the lock files.
		Updates of dependencies should be in separate commits.
	`

	return &OpenAI{
		apiKey: apiKey,
		url:    "https://api.openai.com/v1/chat/completions",
		conversation: []Message{
			{
				Role:    "system",
				Content: defaultMsgContent,
			},
		},
	}
}

func (o *OpenAI) GenCommits(gitDiff string) ([]CommitCandidate, error) {
	o.conversation = append(o.conversation, Message{
		Role:    "user",
		Content: gitDiff,
	})

	body := RequestBody{
		Model:    "gpt-3.5-turbo",
		Messages: o.conversation,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", o.url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get response from OpenAI: %s", respBody)
	}

	var response ResponseBody
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response from OpenAI: %w", err)
	}

	o.conversation = append(o.conversation, response.Choices[0].Message)

	var commitCandidates []CommitCandidate
	err = json.Unmarshal([]byte(response.Choices[0].Message.Content), &commitCandidates)
	if err != nil {
		return nil, fmt.Errorf("failed to parse commit candidates: %w", err)
	}

	return commitCandidates, nil
}
