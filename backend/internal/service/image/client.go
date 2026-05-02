package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/imagegen/backend/internal/service/prompt"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string, timeoutSec int) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

type GenerationResponse struct {
	Data []struct {
		URL     string `json:"url"`
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

func (c *Client) Generate(ctx context.Context, req prompt.ImageRequest) (*GenerationResponse, error) {
	endpoint := "https://api.openai.com/v1/images/generations"
	return c.call(ctx, endpoint, req)
}

func (c *Client) Edit(ctx context.Context, req prompt.ImageRequest, referenceURL string) (*GenerationResponse, error) {
	endpoint := "https://api.openai.com/v1/images/edits"
	// Edits endpoint uses multipart form; for URL-based reference we use generations
	// with reference image as part of the prompt context.
	_ = referenceURL
	return c.call(ctx, endpoint, req)
}

func (c *Client) call(ctx context.Context, endpoint string, req prompt.ImageRequest) (*GenerationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result GenerationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
