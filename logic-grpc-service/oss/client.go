package oss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type Client struct {
	client       *cos.Client
	secretID     string
	secretKey    string
	presignCache *PresignCache
}

func (c *Client) SetPresignCache(cache *PresignCache) { c.presignCache = cache }

func NewClient(endpoint, secretID, secretKey, bucketName, publicBaseURL string) (*Client, error) {
	baseURL := strings.TrimRight(publicBaseURL, "/")
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s.%s", bucketName, endpoint)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse cos public base url: %w", err)
	}
	c := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})
	return &Client{client: c, secretID: secretID, secretKey: secretKey}, nil
}

func (c *Client) GeneratePresignedPutURL(ossKey string) (string, time.Time, error) {
	return c.signURL(context.Background(), http.MethodPut, ossKey)
}

func (c *Client) GeneratePresignedGetURL(ossKey string) (string, error) {
	if c.presignCache != nil {
		if cached, ok := c.presignCache.Get(context.Background(), ossKey); ok {
			return cached, nil
		}
	}
	signed, _, err := c.signURL(context.Background(), http.MethodGet, ossKey)
	if err != nil {
		return "", err
	}
	if c.presignCache != nil {
		c.presignCache.Set(context.Background(), ossKey, signed)
	}
	return signed, err
}

func (c *Client) VerifyObject(ctx context.Context, ossKey string) error {
	if strings.TrimSpace(ossKey) == "" {
		return fmt.Errorf("oss key is empty")
	}
	_, err := c.client.Object.Head(ctx, ossKey, nil)
	return err
}

func (c *Client) DownloadObject(ctx context.Context, ossKey string) ([]byte, error) {
	if strings.TrimSpace(ossKey) == "" {
		return nil, fmt.Errorf("oss key is empty")
	}
	resp, err := c.client.Object.Get(ctx, ossKey, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Limit download size to prevent OOM (25 MB hard cap).
	return io.ReadAll(io.LimitReader(resp.Body, MaxResumeDownloadSizeBytes))
}

func IsNotFound(err error) bool {
	return cos.IsNotFoundError(err)
}

func (c *Client) signURL(ctx context.Context, method, ossKey string) (string, time.Time, error) {
	if c.client == nil {
		return "", time.Time{}, fmt.Errorf("cos client is nil")
	}
	if strings.TrimSpace(ossKey) == "" {
		return "", time.Time{}, fmt.Errorf("oss key is empty")
	}
	expireAt := time.Now().Add(15 * time.Minute)
	presignedURL, err := c.client.Object.GetPresignedURL(ctx, method, ossKey, c.secretID, c.secretKey, 15*time.Minute, nil)
	if err != nil {
		return "", time.Time{}, err
	}
	return presignedURL.String(), expireAt, nil
}
