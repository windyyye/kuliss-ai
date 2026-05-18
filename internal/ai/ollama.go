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
	BaseURL  string
	Model    string
	Brain    *brain.Brain
	Provider string
	ApiKey   string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewClient(baseURL, model string, b *brain.Brain) *Client {
	return &Client{BaseURL: baseURL, Model: model, Brain: b}
}

func NewProviderClient(provider, baseURL, model, apiKey string, b *brain.Brain) *Client {
	return &Client{
		BaseURL:  baseURL,
		Model:    model,
		Brain:    b,
		Provider: provider,
		ApiKey:   apiKey,
	}
}

var thinkRegex = regexp.MustCompile(`(?s)<\|channel>.*?<channel\|>`)
var thinkRegexAlt = regexp.MustCompile(`(?s)<think\b[^>]*>.*?</think\s*>`)

func stripThinkingTokens(s string) string {
	s = thinkRegex.ReplaceAllString(s, "")
	s = thinkRegexAlt.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

var httpClient = &http.Client{Timeout: 90 * time.Second}

func (c *Client) Chat(history []map[string]string, userMessage string) (string, error) {
	messages := c.buildMessages(history, userMessage)

	if c.Provider == "openrouter" {
		return c.chatOpenRouter(messages)
	}
	return c.chatOllama(messages)
}

func (c *Client) buildMessages(history []map[string]string, userMessage string) []Message {
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

	messages := []Message{{Role: "system", Content: systemPrompt}}

	for _, m := range history {
		messages = append(messages, Message{Role: m["role"], Content: m["content"]})
	}

	messages = append(messages, Message{Role: "user", Content: userMessage})
	return messages
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []Message       `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  ollamaChatOpts  `json:"options"`
}

type ollamaChatOpts struct {
	NumPredict  int     `json:"num_predict"`
	Temperature float64 `json:"temperature"`
	TopK        int     `json:"top_k"`
	TopP        float64 `json:"top_p"`
}

type ollamaChatResponse struct {
	Message         Message `json:"message"`
	PromptTokens    int     `json:"prompt_eval_count"`
	CompletionTokens int    `json:"eval_count"`
}

func (c *Client) chatOllama(messages []Message) (string, error) {
	reqBody, err := json.Marshal(ollamaChatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   false,
		Options: ollamaChatOpts{
			NumPredict:  500,
			Temperature: 0.4,
			TopK:        40,
			TopP:        0.9,
		},
	})
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}

	baseURL := strings.TrimRight(c.BaseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1")
	baseURL = strings.TrimRight(baseURL, "/")

	resp, err := httpClient.Post(baseURL+"/api/chat", "application/json", bytes.NewBuffer(reqBody))
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

	var chatResp ollamaChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		log.Printf("[AI DEBUG] Unmarshal hatası. Ham yanıt: %s", string(body))
		return "", fmt.Errorf("json unmarshal: %w", err)
	}

	if chatResp.Message.Content == "" {
		log.Printf("[AI DEBUG] Model boş cevap döndürdü. Ham yanıt: %s", string(body))
	}

	log.Printf("[AI] provider=ollama model=%s prompt_tokens=%d completion_tokens=%d",
		c.Model, chatResp.PromptTokens, chatResp.CompletionTokens)

	return stripThinkingTokens(chatResp.Message.Content), nil
}

type openRouterRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	TopP        float64   `json:"top_p"`
	Stream      bool      `json:"stream"`
}

type openRouterResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *Client) chatOpenRouter(messages []Message) (string, error) {
	reqBody, err := json.Marshal(openRouterRequest{
		Model:       c.Model,
		Messages:    messages,
		MaxTokens:   500,
		Temperature: 0.4,
		TopP:        0.9,
		Stream:      false,
	})
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}

	endpoint := strings.TrimRight(c.BaseURL, "/")
	if !strings.Contains(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("request oluşturma: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.ApiKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openrouter isteği: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("yanıt okuma: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenRouter Hatası (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var orResp openRouterResponse
	if err := json.Unmarshal(body, &orResp); err != nil {
		log.Printf("[AI DEBUG] OpenRouter unmarshal hatası. Ham yanıt: %s", string(body))
		return "", fmt.Errorf("json unmarshal: %w", err)
	}

	if len(orResp.Choices) == 0 || orResp.Choices[0].Message.Content == "" {
		log.Printf("[AI DEBUG] Model boş cevap döndürdü. Ham yanıt: %s", string(body))
		return "", nil
	}

	log.Printf("[AI] provider=openrouter model=%s prompt_tokens=%d completion_tokens=%d total_tokens=%d",
		c.Model, orResp.Usage.PromptTokens, orResp.Usage.CompletionTokens, orResp.Usage.TotalTokens)

	return stripThinkingTokens(orResp.Choices[0].Message.Content), nil
}
