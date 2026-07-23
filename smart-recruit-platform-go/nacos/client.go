package nacos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPClient struct {
	addresses []*url.URL
	client    *http.Client
	username  string
	password  string
}

type ClientOptions struct {
	Addresses string
	Username  string
	Password  string
	Timeout   time.Duration
	Client    *http.Client
}

func NewHTTPClient(options ClientOptions) (*HTTPClient, error) {
	addresses, err := ParseAddresses(options.Addresses)
	if err != nil {
		return nil, err
	}
	client := options.Client
	if client == nil {
		timeout := options.Timeout
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	return &HTTPClient{
		addresses: addresses,
		client:    client,
		username:  strings.TrimSpace(options.Username),
		password:  strings.TrimSpace(options.Password),
	}, nil
}

func (c *HTTPClient) get(ctx context.Context, apiPath string, values url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodGet, apiPath, values)
}

func (c *HTTPClient) post(ctx context.Context, apiPath string, values url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodPost, apiPath, values)
}

func (c *HTTPClient) delete(ctx context.Context, apiPath string, values url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodDelete, apiPath, values)
}

func (c *HTTPClient) do(ctx context.Context, method string, apiPath string, values url.Values) ([]byte, error) {
	if c == nil || len(c.addresses) == 0 {
		return nil, errors.New("nacos client is not configured")
	}
	var lastErr error
	for _, base := range c.addresses {
		endpoint := *base
		endpoint.Path = strings.TrimRight(base.Path, "/") + apiPath
		endpoint.RawQuery = values.Encode()
		req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), nil)
		if err != nil {
			return nil, err
		}
		if c.username != "" || c.password != "" {
			req.SetBasicAuth(c.username, c.password)
		}
		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}
		lastErr = fmt.Errorf("nacos %s %s returned %d: %s", method, apiPath, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if lastErr == nil {
		lastErr = errors.New("nacos request failed")
	}
	return nil, lastErr
}
