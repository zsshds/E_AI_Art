package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/repo"
)

type Client struct {
	apiKey      string
	baseURL     string
	settingRepo *repo.SettingRepo
	httpClient  *http.Client
}

type resolvedConfig struct {
	baseURL             string
	apiKey              string
	generationURL       string
	editURL             string
	pollURLPrefix       string
	bananaGenerationURL string
	chatURL             string
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

func (r *TaskResultResponse) UnmarshalJSON(data []byte) error {
	type taskResultResponse TaskResultResponse
	var aux struct {
		Progress json.RawMessage `json:"progress"`
		*taskResultResponse
	}
	aux.taskResultResponse = (*taskResultResponse)(r)

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Progress) == 0 || string(aux.Progress) == "null" {
		return nil
	}

	progress, err := parseProgress(aux.Progress)
	if err != nil {
		return fmt.Errorf("parse progress: %w", err)
	}
	r.Progress = progress
	return nil
}

func parseProgress(raw json.RawMessage) (int, error) {
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
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
func (c *Client) resolveConfig(ctx context.Context) resolvedConfig {
	baseURL := c.baseURL
	apiKey := c.apiKey
	genPath := "/v1/images/generations/tasks"
	editPath := "/v1/images/edits"
	pollPath := "/v1/images/tasks/"
	bananaPath := "/v1/images/generations"
	chatPath := "/v1/chat/completions"
	genURL := ""
	editURL := ""
	pollURL := ""
	bananaURL := ""
	chatURL := ""

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
		if v, err := c.settingRepo.Get(ctx, "api_edit_path"); err == nil && v != "" {
			editPath = v
		}
		if v, err := c.settingRepo.Get(ctx, "api_poll_path"); err == nil && v != "" {
			pollPath = v
		}
		if v, err := c.settingRepo.Get(ctx, "api_chat_path"); err == nil && v != "" {
			chatPath = v
		}
		if v, err := c.settingRepo.Get(ctx, "api_generation_url"); err == nil && v != "" {
			genURL = v
		}
		if v, err := c.settingRepo.Get(ctx, "api_edit_url"); err == nil && v != "" {
			editURL = v
		}
		if v, err := c.settingRepo.Get(ctx, "api_poll_url"); err == nil && v != "" {
			pollURL = v
		}
		if v, err := c.settingRepo.Get(ctx, "api_chat_url"); err == nil && v != "" {
			chatURL = v
		}
		if v, err := c.settingRepo.Get(ctx, "api_banana_generation_url"); err == nil && v != "" {
			bananaURL = v
		}
	}

	return resolvedConfig{
		baseURL:             strings.TrimSpace(baseURL),
		apiKey:              apiKey,
		generationURL:       pickConfiguredURL(baseURL, genURL, genPath),
		editURL:             pickConfiguredURL(baseURL, editURL, editPath),
		pollURLPrefix:       pickConfiguredURL(baseURL, pollURL, pollPath),
		bananaGenerationURL: pickConfiguredURL(baseURL, bananaURL, bananaPath),
		chatURL:             pickConfiguredURL(baseURL, chatURL, chatPath),
	}
}

// CreateImageTask submits an async image generation task.
func (c *Client) CreateImageTask(ctx context.Context, body map[string]interface{}) (*GenTaskResponse, error) {
	cfg := c.resolveConfig(ctx)
	modelID, _ := body["model"].(string)
	if model.GetModelType(modelID) == model.ModelTypeBanana {
		return c.postTask(ctx, cfg.bananaGenerationURL, cfg.apiKey, body)
	}
	return c.postTask(ctx, cfg.generationURL, cfg.apiKey, body)
}

// EditImageTask submits an async image edit task.
func (c *Client) EditImageTask(ctx context.Context, body map[string]interface{}) (*GenTaskResponse, error) {
	cfg := c.resolveConfig(ctx)
	payload, contentType, err := buildEditMultipartBody(ctx, c.httpClient, body)
	if err != nil {
		return nil, err
	}

	log.Printf("[IMG-API] POST %s", cfg.editURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.editURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
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

// ChatCompletion sends a chat completions request. Used when source images are present
// so the vision-capable model can "read" image content before generating.
func (c *Client) ChatCompletion(ctx context.Context, body map[string]interface{}) (*GenTaskResponse, error) {
	cfg := c.resolveConfig(ctx)
	return c.postTask(ctx, cfg.chatURL, cfg.apiKey, body)
}

// GetTaskResult polls for the result of an async task (skip for sync models).
func (c *Client) GetTaskResult(ctx context.Context, taskID string) (*TaskResultResponse, error) {
	cfg := c.resolveConfig(ctx)
	return c.getTask(ctx, joinURLPath(cfg.pollURLPrefix, taskID), cfg.apiKey)
}

// ListModels fetches available models from the AI platform.
func (c *Client) ListModels(ctx context.Context, path string) (*ListModelsResponse, error) {
	cfg := c.resolveConfig(ctx)
	return c.getModels(ctx, pickConfiguredURL(cfg.baseURL, "", path), cfg.apiKey)
}

func pickConfiguredURL(baseURL, fullURL, path string) string {
	fullURL = strings.TrimSpace(fullURL)
	if fullURL != "" {
		return fullURL
	}
	return joinURLPath(baseURL, path)
}

func joinURLPath(baseURL, path string) string {
	baseURL = strings.TrimSpace(baseURL)
	path = strings.TrimSpace(path)
	if path == "" {
		return strings.TrimRight(baseURL, "/")
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if baseURL == "" {
		if strings.HasPrefix(path, "/") {
			return path
		}
		return "/" + path
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
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

func buildEditMultipartBody(ctx context.Context, httpClient *http.Client, body map[string]interface{}) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for _, key := range []string{"model", "prompt", "size", "background", "output_format", "quality"} {
		if value, ok := body[key]; ok {
			if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
				if err := writer.WriteField(key, s); err != nil {
					return nil, "", fmt.Errorf("write field %s: %w", key, err)
				}
			}
		}
	}
	if value, ok := body["n"]; ok {
		switch n := value.(type) {
		case int:
			if err := writer.WriteField("n", strconv.Itoa(n)); err != nil {
				return nil, "", fmt.Errorf("write field n: %w", err)
			}
		case float64:
			if err := writer.WriteField("n", strconv.Itoa(int(n))); err != nil {
				return nil, "", fmt.Errorf("write field n: %w", err)
			}
		}
	}

	imageSources := collectEditImageSources(body)
	if len(imageSources) == 0 {
		return nil, "", fmt.Errorf("image edit request requires at least one image")
	}

	for _, source := range imageSources {
		filename, content, contentType, err := loadImageSource(ctx, httpClient, source)
		if err != nil {
			return nil, "", err
		}
		part, err := writer.CreateFormFile("image", filename)
		if err != nil {
			return nil, "", fmt.Errorf("create form file: %w", err)
		}
		if _, err := part.Write(content); err != nil {
			return nil, "", fmt.Errorf("write form file: %w", err)
		}
		_ = contentType
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer: %w", err)
	}

	return buf.Bytes(), writer.FormDataContentType(), nil
}

func collectEditImageSources(body map[string]interface{}) []string {
	sources := make([]string, 0, 8)
	appendString := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			sources = append(sources, value)
		}
	}
	appendSlice := func(items []string) {
		for _, item := range items {
			appendString(item)
		}
	}

	if image, ok := body["image"].(string); ok {
		appendString(image)
	}
	if images, ok := body["images"].([]string); ok {
		appendSlice(images)
	}
	if images, ok := body["images"].([]interface{}); ok {
		for _, item := range images {
			if s, ok := item.(string); ok {
				appendString(s)
			}
		}
	}
	if refs, ok := body["reference_images"].([]string); ok {
		appendSlice(refs)
	}
	if refs, ok := body["reference_images"].([]interface{}); ok {
		for _, item := range refs {
			if s, ok := item.(string); ok {
				appendString(s)
			}
		}
	}
	return sources
}

func loadImageSource(ctx context.Context, httpClient *http.Client, source string) (string, []byte, string, error) {
	source = strings.TrimSpace(source)
	if strings.HasPrefix(source, "data:") {
		return decodeDataURL(source)
	}
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return "", nil, "", fmt.Errorf("new image request: %w", err)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return "", nil, "", fmt.Errorf("download image: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			return "", nil, "", fmt.Errorf("download image status %d: %s", resp.StatusCode, string(body))
		}
		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", nil, "", fmt.Errorf("read image body: %w", err)
		}
		contentType := resp.Header.Get("Content-Type")
		filename := path.Base(strings.Split(source, "?")[0])
		if filename == "" || filename == "." || filename == "/" {
			filename = "image.png"
		}
		if contentType == "" {
			contentType = "image/png"
		}
		return filename, content, contentType, nil
	}
	return "", nil, "", fmt.Errorf("unsupported image source: %s", source)
}

func decodeDataURL(value string) (string, []byte, string, error) {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return "", nil, "", fmt.Errorf("invalid data url")
	}
	meta := parts[0]
	payload := parts[1]
	if !strings.Contains(meta, ";base64") {
		return "", nil, "", fmt.Errorf("unsupported data url encoding")
	}

	contentType := strings.TrimPrefix(strings.SplitN(meta, ";", 2)[0], "data:")
	if contentType == "" {
		contentType = "image/png"
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", nil, "", fmt.Errorf("decode data url: %w", err)
	}

	ext := ".png"
	switch contentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	}

	return "upload" + ext, raw, contentType, nil
}
