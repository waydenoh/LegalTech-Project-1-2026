package llm

import (
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"strings"
"time"
)

type OllamaClient struct {
BaseURL    string
HTTPClient *http.Client
}

func NewOllamaClient(baseURL string) *OllamaClient {
if baseURL == "" {
baseURL = "http://localhost:11434"
}
return &OllamaClient{
BaseURL: strings.TrimRight(baseURL, "/"),
HTTPClient: &http.Client{
Timeout: 5 * time.Minute,
},
}
}

type chatMessage struct {
Role    string `json:"role"`
Content string `json:"content"`
}

type chatRequest struct {
Model    string                 `json:"model"`
Messages []chatMessage          `json:"messages"`
Stream   bool                   `json:"stream"`
Think    bool                   `json:"think"`
Format   map[string]any         `json:"format"`
Options  map[string]interface{} `json:"options,omitempty"`
}

type chatResponse struct {
Message struct {
Role    string `json:"role"`
Content string `json:"content"`
} `json:"message"`
DoneReason string `json:"done_reason,omitempty"`
Error      string `json:"error,omitempty"`
}

type tagsResponse struct {
Models []struct {
Name string `json:"name"`
} `json:"models"`
}

func (c *OllamaClient) Health(ctx context.Context) ([]string, error) {
req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/tags", nil)
if err != nil {
return nil, err
}
resp, err := c.HTTPClient.Do(req)
if err != nil {
return nil, fmt.Errorf("ollama unreachable at %s: %w", c.BaseURL, err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
}

var tags tagsResponse
if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
return nil, err
}

names := make([]string, 0, len(tags.Models))
for _, m := range tags.Models {
names = append(names, m.Name)
}
return names, nil
}

func (c *OllamaClient) GenerateStructured(
ctx context.Context,
model string,
systemPrompt string,
userPrompt string,
schema map[string]any,
out any,
) error {
payload := chatRequest{
Model: model,
Messages: []chatMessage{
{Role: "system", Content: systemPrompt},
{Role: "user", Content: userPrompt},
},
Stream: false,
Think:  false,
Format: schema,
Options: map[string]interface{}{
"temperature": 0.1,
"num_ctx":     8192,
"num_predict": 4096,
},
}

body, err := json.Marshal(payload)
if err != nil {
return fmt.Errorf("marshal request: %w", err)
}

req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/chat", bytes.NewReader(body))
if err != nil {
return fmt.Errorf("create request: %w", err)
}
req.Header.Set("Content-Type", "application/json")

resp, err := c.HTTPClient.Do(req)
if err != nil {
return fmt.Errorf("call ollama: %w", err)
}
defer resp.Body.Close()

rawResp, err := io.ReadAll(resp.Body)
if err != nil {
return fmt.Errorf("read response: %w", err)
}

if resp.StatusCode != http.StatusOK {
return fmt.Errorf("ollama HTTP %d: %s", resp.StatusCode, string(rawResp))
}

var parsed chatResponse
if err := json.Unmarshal(rawResp, &parsed); err != nil {
return fmt.Errorf("decode chat envelope: %w", err)
}
if parsed.Error != "" {
return fmt.Errorf("ollama error: %s", parsed.Error)
}

content := strings.TrimSpace(parsed.Message.Content)
if content == "" {
return fmt.Errorf("ollama returned empty structured response (done_reason: %s)", parsed.DoneReason)
}

if err := json.Unmarshal([]byte(content), out); err != nil {
if parsed.DoneReason == "length" {
return fmt.Errorf("model output hit token limit and truncated JSON (done_reason=length): %w", err)
}
return fmt.Errorf("unmarshal structured output (%s): %w", content, err)
}
return nil
}
