package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"smart-recruit-platform-go/logger"
)

const (
	DefaultBailianModel      = "text-embedding-v4"
	DefaultBailianTimeout    = 30 * time.Second
	DefaultBailianMaxRetries = 2
	bailianContentType       = "application/json"
	bailianAuthScheme        = "Bearer"
)

type bailianEmbedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type bailianEmbedResponse struct {
	Output    *bailianOutput `json:"output,omitempty"`
	Data      []bailianData  `json:"data,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

type bailianOutput struct {
	Embeddings []bailianEmbedding `json:"embeddings"`
}

type bailianData struct {
	Embedding []float64 `json:"embedding"`
}

type bailianEmbedding struct {
	Embedding []float64 `json:"embedding"`
}

type BailianTextEmbeddingProvider struct {
	endpoint   string
	apiKey     string
	model      string
	dimensions int
	httpClient *http.Client
	maxRetries int
}

type BailianTextEmbeddingProviderOption func(*BailianTextEmbeddingProvider)

func WithBailianTimeout(timeout time.Duration) BailianTextEmbeddingProviderOption {
	return func(p *BailianTextEmbeddingProvider) {
		if timeout > 0 {
			p.httpClient.Timeout = timeout
		}
	}
}

func WithBailianMaxRetries(retries int) BailianTextEmbeddingProviderOption {
	return func(p *BailianTextEmbeddingProvider) {
		if retries >= 0 {
			p.maxRetries = retries
		}
	}
}

func WithBailianModel(model string) BailianTextEmbeddingProviderOption {
	return func(p *BailianTextEmbeddingProvider) {
		if model != "" {
			p.model = model
		}
	}
}

func WithBailianDimensions(dimensions int) BailianTextEmbeddingProviderOption {
	return func(p *BailianTextEmbeddingProvider) {
		if dimensions > 0 {
			p.dimensions = dimensions
		}
	}
}

func NewBailianTextEmbeddingProvider(endpoint, apiKey string, opts ...BailianTextEmbeddingProviderOption) *BailianTextEmbeddingProvider {
	p := &BailianTextEmbeddingProvider{
		endpoint:   endpoint,
		apiKey:     apiKey,
		model:      DefaultBailianModel,
		httpClient: &http.Client{Timeout: DefaultBailianTimeout},
		maxRetries: DefaultBailianMaxRetries,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *BailianTextEmbeddingProvider) EmbedText(ctx context.Context, text string) (EmbeddingVector, error) {
	log := logger.GetRequestLogger(ctx)

	body := bailianEmbedRequest{
		Model:      p.model,
		Input:      []string{text},
		Dimensions: p.dimensions,
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return EmbeddingVector{}, fmt.Errorf("bailian: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return EmbeddingVector{}, fmt.Errorf("bailian: create request: %w", err)
	}
	req.Header.Set("Content-Type", bailianContentType)
	req.Header.Set("Authorization", bailianAuthScheme+" "+p.apiKey)

	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			log.Warn("[bailian] retrying embedding request",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", p.maxRetries),
				zap.Error(lastErr))
			time.Sleep(backoffDuration(attempt))
		}

		resp, err := p.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("bailian: request failed: %w", err)
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			if readErr != nil {
				return EmbeddingVector{}, fmt.Errorf("bailian: read response: %w", readErr)
			}
			result, parseErr := parseBailianResponse(respBody)
			if parseErr != nil {
				return EmbeddingVector{}, parseErr
			}
			return result, nil
		}

		lastErr = fmt.Errorf("bailian: status %d", resp.StatusCode)
		if readErr == nil && len(respBody) > 0 {
			lastErr = fmt.Errorf("bailian: status %d: %s", resp.StatusCode, truncateString(string(respBody), 200))
		}

		if !isRetryableStatus(resp.StatusCode) {
			log.Warn("[bailian] non-retryable status",
				zap.Int("status", resp.StatusCode),
				zap.Error(lastErr))
			return EmbeddingVector{}, lastErr
		}
	}

	return EmbeddingVector{}, fmt.Errorf("bailian: max retries exhausted: %w", lastErr)
}

func parseBailianResponse(body []byte) (EmbeddingVector, error) {
	var resp bailianEmbedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return EmbeddingVector{}, fmt.Errorf("bailian: parse response: %w", err)
	}

	var vector []float64

	if resp.Output != nil && len(resp.Output.Embeddings) > 0 {
		vector = resp.Output.Embeddings[0].Embedding
	} else if len(resp.Data) > 0 {
		vector = resp.Data[0].Embedding
	}

	if len(vector) == 0 {
		return EmbeddingVector{}, fmt.Errorf("bailian: empty embedding vector in response: %w", ErrEmbeddingVectorInvalid)
	}

	if err := validateEmbeddingVector(vector); err != nil {
		return EmbeddingVector{}, fmt.Errorf("bailian: %w", err)
	}

	return EmbeddingVector{
		Model:  extractModelFromResponse(body),
		Vector: vector,
	}, nil
}

func extractModelFromResponse(body []byte) string {
	var raw struct {
		Model string `json:"model,omitempty"`
	}
	if err := json.Unmarshal(body, &raw); err == nil && raw.Model != "" {
		return raw.Model
	}
	return DefaultBailianModel
}

// Name implements EmbeddingProvider. Bailian-backed providers always report
// "bailian" regardless of underlying model variant; the model name is
// exposed separately via the EmbeddingVector.Model field.
func (p *BailianTextEmbeddingProvider) Name() string { return "bailian" }

func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusRequestTimeout ||
		statusCode >= 500
}

func backoffDuration(attempt int) time.Duration {
	return time.Duration(attempt*100) * time.Millisecond
}
