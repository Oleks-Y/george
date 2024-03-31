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
	CommitMessage string             `json:"commitMessage"`
	Files         []FilePatchRequest `json:"files"`
}

func NewOpenAI() *OpenAI {
	apiKey := os.Getenv("GEORGE_OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEORGE_OPENAI_API_KEY is not set. Please set it and try again.")
	}

	defaultPrompt, err := os.ReadFile("./prompt.txt")
	if err != nil {
		log.Fatal("failed to read prompt.txt: ", err)
	}

	defaultMsgContent := string(defaultPrompt)

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

func (o *OpenAI) GenCommits(gitDiff GitDiff) ([]CommitCandidate, error) {
	gitDiffJson, err := json.MarshalIndent(gitDiff, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal git diff: %w", err)
	}

	diffStr := string(gitDiffJson)

	o.conversation = append(o.conversation, Message{
		Role:    "user",
		Content: diffStr,
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
