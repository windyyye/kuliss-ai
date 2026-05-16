package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"message-go/internal/brain"
)

type Client struct {
	BaseURL	string
	Model	string
	Brain	*brain.Brain
}

type chatRequest struct {
	Model		string		`json:"model"`
	Messages	[]Message	`json:"messages"`
	Stream		bool		`json:"stream"`
	Options		chatOptions	`json:"options"`
}

type chatOptions struct {
	NumPredict	int	`json:"num_predict"`
	Temperature	float64	`json:"temperature"`
	TopK		int	`json:"top_k"`
	TopP		float64	`json:"top_p"`
}

type Message struct {
	Role	string	`json:"role"`
	Content	string	`json:"content"`
}

type chatResponse struct {
	Message Message `json:"message"`
}

func NewClient(baseURL, model string, b *brain.Brain) *Client {
	return &Client{BaseURL: baseURL, Model: model, Brain: b}
}

var thinkRegex = regexp.MustCompile(`(?s)<\|channel>.*?<channel\|>`)

func stripThinkingTokens(s string) string {
	s = thinkRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

var httpClient = &http.Client{Timeout: 90 * time.Second}

func (c *Client) Chat(history []map[string]string, userMessage string) (string, error) {

	var messages []Message

	var systemPrompt string
	if c.Brain != nil {
		systemPrompt = c.Brain.AssemblePrompt(userMessage)
	}
	if systemPrompt == "" {
		execPath, _ := os.Executable()
		promptPath := filepath.Join(filepath.Dir(execPath), "prompt.txt")

		promptBytes, err := os.ReadFile(promptPath)
		if err != nil {
			promptBytes, err = os.ReadFile("prompt.txt")
		}

		systemPrompt = "Sen bir yapay zeka asistanısın."
		if err == nil {
			systemPrompt = string(promptBytes)
		}
	}

	messages = append(messages, Message{
		Role:		"system",
		Content:	systemPrompt,
	})

	for _, m := range history {
		messages = append(messages, Message{
			Role:		m["role"],
			Content:	m["content"],
		})
	}

	messages = append(messages, Message{
		Role:		"user",
		Content:	userMessage,
	})

	reqBody, err := json.Marshal(chatRequest{
		Model:		c.Model,
		Messages:	messages,
		Stream:		false,
		Options: chatOptions{
			NumPredict:	1500,
			Temperature:	0.7,
			TopK:		64,
			TopP:		0.95,
		},
	})
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}

	baseURL := strings.TrimRight(c.BaseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1")
	baseURL = strings.TrimRight(baseURL, "/")

	resp, err := httpClient.Post(
		baseURL+"/api/chat",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return "", fmt.Errorf("ollama isteği: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("yanıt okuma: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama Hatası (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		log.Printf("[AI DEBUG] Unmarshal hatası. Ham yanıt: %s", string(body))
		return "", fmt.Errorf("json unmarshal: %w", err)
	}

	if chatResp.Message.Content == "" {
		log.Printf("[AI DEBUG] Model boş cevap döndürdü. Ham yanıt: %s", string(body))
	}

	clean := stripThinkingTokens(chatResp.Message.Content)
	return clean, nil
}
