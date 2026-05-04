package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/repo"
)

type Client struct {
	apiKey     string
	baseURL    string
	settingRepo *repo.SettingRepo
	httpClient  *http.Client
}

// --- Response types ---

type GenTaskResponse struct {
	Created int              `json:"created"`
	Data    []GenTaskData    `json:"data"`
	Tasks   []GenTaskSummary `json:"tasks"`
	// Chat completions format
	Choices []ChatChoice `json:"choices,omitempty"`
}

type ChatChoice struct {
	Message ChatMessage `json:"message"`
}

type ChatMessage struct {
	Content string `json:"content"`
}

type GenTaskData struct {
	URL     string `json:"url"`
	B64JSON string `json:"b64_json"`
}

type GenTaskSummary struct {
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
	Status string `json:"status"`
}

type TaskResultResponse struct {
	TaskID     string        `json:"task_id"`
	Status     string        `json:"status"`
	Progress   int           `json:"progress"`
	FailReason string        `json:"fail_reason"`
	Data       []GenTaskData `json:"data"`
	Usage      *TaskUsage    `json:"usage,omitempty"`
}

type TaskUsage struct {
	TotalTokens        int                 `json:"total_tokens"`
	InputTokens        int                 `json:"input_tokens"`
	OutputTokens       int                 `json:"output_tokens"`
	InputTokensDetails *InputTokensDetails `json:"input_tokens_details,omitempty"`
}

type InputTokensDetails struct {
	TextTokens  int `json:"text_tokens"`
	ImageTokens int `json:"image_tokens"`
}

// --- Model list types ---

type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ListModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// --- Task status constants ---
const (
	TaskStatusPending    = "PENDING"
	TaskStatusNotStart   = "NOT_START"
	TaskStatusInProgress = "IN_PROGRESS"
	TaskStatusSuccess    = "SUCCESS"
	TaskStatusFailure    = "FAILURE"
)

func IsTerminalStatus(status string) bool {
	return status == TaskStatusSuccess || status == TaskStatusFailure
}

func IsRunningStatus(status string) bool {
	return status == TaskStatusPending || status == TaskStatusNotStart || status == TaskStatusInProgress
}

func NewClient(apiKey, baseURL string, settingRepo *repo.SettingRepo, timeoutSec int) *Client {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &Client{
		apiKey:      apiKey,
		baseURL:     baseURL,
		settingRepo: settingRepo,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

// resolveConfig reads dynamic settings from DB, falling back to config defaults.
func (c *Client) resolveConfig(ctx context.Context) (baseURL, apiKey, genPath, pollPath, bananaPath string) {
	baseURL = c.baseURL
	apiKey = c.apiKey
	genPath = "/v1/images/generations/tasks"
	pollPath = "/v1/images/tasks/"
	bananaPath = "/v1/images/generations"

	if c.settingRepo != nil {
		if v, err := c.settingRepo.Get(ctx, "api_base_url"); err == nil && v != "" {
			baseURL = v
		}
		if v, err := c.settingRepo.Get(ctx, "api_key"); err == nil && v != "" {
			apiKey = v
		}
		if v, err := c.settingRepo.Get(ctx, "api_generation_path"); err == nil && v != "" {
			genPath = v
		}
		if v, err := c.settingRepo.Get(ctx, "api_poll_path"); err == nil && v != "" {
			pollPath = v
		}
	}
	return
}

// CreateImageTask submits an async image generation task.
func (c *Client) CreateImageTask(ctx context.Context, body map[string]interface{}) (*GenTaskResponse, error) {
	baseURL, apiKey, genPath, _, bananaPath := c.resolveConfig(ctx)
	modelID, _ := body["model"].(string)
	if model.GetModelType(modelID) == model.ModelTypeBanana {
		return c.postTask(ctx, baseURL+bananaPath, apiKey, body)
	}
	return c.postTask(ctx, baseURL+genPath, apiKey, body)
}

// EditImageTask submits an async image edit task.
func (c *Client) EditImageTask(ctx context.Context, body map[string]interface{}) (*GenTaskResponse, error) {
	baseURL, apiKey, genPath, _, bananaPath := c.resolveConfig(ctx)
	modelID, _ := body["model"].(string)
	if model.GetModelType(modelID) == model.ModelTypeBanana {
		return c.postTask(ctx, baseURL+bananaPath, apiKey, body)
	}
	return c.postTask(ctx, baseURL+genPath, apiKey, body)
}

// GetTaskResult polls for the result of an async task (skip for sync models).
func (c *Client) GetTaskResult(ctx context.Context, taskID string) (*TaskResultResponse, error) {
	baseURL, apiKey, _, pollPath, _ := c.resolveConfig(ctx)
	return c.getTask(ctx, baseURL+pollPath+taskID, apiKey)
}

// ListModels fetches available models from the AI platform.
func (c *Client) ListModels(ctx context.Context, path string) (*ListModelsResponse, error) {
	baseURL, apiKey, _, _, _ := c.resolveConfig(ctx)
	fullURL := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		fullURL = baseURL + path
	}
	return c.getModels(ctx, fullURL, apiKey)
}

func (c *Client) getModels(ctx context.Context, fullURL, apiKey string) (*ListModelsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	var result ListModelsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}
	return &result, nil
}

func (c *Client) postTask(ctx context.Context, fullURL, apiKey string, body map[string]interface{}) (*GenTaskResponse, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	log.Printf("[IMG-API] POST %s", fullURL)
	log.Printf("[IMG-API] Request: %s", string(data))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	log.Printf("[IMG-API] Auth: Bearer %s...", apiKey[:min(8, len(apiKey))])

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[IMG-API] ERROR: %v", err)
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	log.Printf("[IMG-API] Response status=%d body=%s", resp.StatusCode, string(respBody))

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	var result GenTaskResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}
	return &result, nil
}

func (c *Client) getTask(ctx context.Context, fullURL, apiKey string) (*TaskResultResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[IMG-API] GET %s ERROR: %v", fullURL, err)
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	log.Printf("[IMG-API] GET %s -> status=%d progress=%s", fullURL, resp.StatusCode, tryExtractProgress(respBody))

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	var result TaskResultResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}
	return &result, nil
}

func tryExtractProgress(body []byte) string {
	var m map[string]interface{}
	if json.Unmarshal(body, &m) == nil {
		if status, ok := m["status"]; ok {
			progress := ""
			if p, ok := m["progress"]; ok {
				progress = fmt.Sprintf(" progress=%v", p)
			}
			return fmt.Sprintf("%v%s", status, progress)
		}
	}
	return "?"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
