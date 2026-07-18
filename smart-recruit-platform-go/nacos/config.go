package nacos

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

type ConfigProvider interface {
	Load(ctx context.Context, dataID string) (string, error)
}

type ConfigOptions struct {
	Client         *HTTPClient
	Addresses      string
	Namespace      string
	Group          string
	Env            string
	StaticFallback map[string]string
	AllowFallback  bool
	Username       string
	Password       string
}

type HTTPConfigProvider struct {
	client    *HTTPClient
	namespace string
	group     string
}

type StaticConfigProvider struct {
	values map[string]string
}

func NewConfigProvider(options ConfigOptions) (ConfigProvider, error) {
	group := defaultGroup(options.Group)
	if options.Client != nil {
		return HTTPConfigProvider{client: options.Client, namespace: options.Namespace, group: group}, nil
	}
	if strings.TrimSpace(options.Addresses) != "" {
		client, err := NewHTTPClient(ClientOptions{
			Addresses: options.Addresses,
			Username:  options.Username,
			Password:  options.Password,
		})
		if err != nil {
			return nil, err
		}
		return HTTPConfigProvider{client: client, namespace: options.Namespace, group: group}, nil
	}
	if options.AllowFallback && isLocalEnv(options.Env) {
		return StaticConfigProvider{values: cloneMap(options.StaticFallback)}, nil
	}
	return nil, errors.New("nacos config provider requires NACOS_ADDR outside local fallback")
}

func (p HTTPConfigProvider) Load(ctx context.Context, dataID string) (string, error) {
	if strings.TrimSpace(dataID) == "" {
		return "", errors.New("nacos config dataId is required")
	}
	values := url.Values{}
	values.Set("dataId", dataID)
	values.Set("group", defaultGroup(p.group))
	if strings.TrimSpace(p.namespace) != "" {
		values.Set("tenant", strings.TrimSpace(p.namespace))
	}
	body, err := p.client.get(ctx, "/nacos/v1/cs/configs", values)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (p StaticConfigProvider) Load(_ context.Context, dataID string) (string, error) {
	if strings.TrimSpace(dataID) == "" {
		return "", errors.New("static config dataId is required")
	}
	value, ok := p.values[dataID]
	if !ok {
		return "", errors.New("static config fallback is missing dataId")
	}
	return value, nil
}

func defaultGroup(group string) string {
	if strings.TrimSpace(group) == "" {
		return "DEFAULT_GROUP"
	}
	return strings.TrimSpace(group)
}

func isLocalEnv(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "", "local", "dev", "development", "test":
		return true
	default:
		return false
	}
}

func cloneMap(values map[string]string) map[string]string {
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
