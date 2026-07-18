package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"smart-recruit-platform-go/nacos"
)

type NacosRuntime struct {
	Config    nacos.ConfigProvider
	Discovery nacos.Discovery
	Env       string
}

type NacosOptions struct {
	Env              string
	Addresses        string
	Namespace        string
	Group            string
	StaticConfigs    map[string]string
	StaticTargets    map[string][]nacos.Instance
	AllowLocalStatic bool
}

func NewNacosRuntime(options NacosOptions) (*NacosRuntime, error) {
	configProvider, err := nacos.NewConfigProvider(nacos.ConfigOptions{
		Addresses:      options.Addresses,
		Namespace:      options.Namespace,
		Group:          options.Group,
		Env:            options.Env,
		StaticFallback: options.StaticConfigs,
		AllowFallback:  options.AllowLocalStatic,
	})
	if err != nil {
		return nil, fmt.Errorf("gateway nacos config: %w", err)
	}
	discovery, err := nacos.NewDiscovery(nacos.DiscoveryOptions{
		Addresses:      options.Addresses,
		Namespace:      options.Namespace,
		Group:          options.Group,
		Env:            options.Env,
		StaticFallback: options.StaticTargets,
		AllowFallback:  options.AllowLocalStatic,
	})
	if err != nil {
		return nil, fmt.Errorf("gateway nacos discovery: %w", err)
	}
	return &NacosRuntime{Config: configProvider, Discovery: discovery, Env: options.Env}, nil
}

func (runtime *NacosRuntime) LoadRequiredConfig(ctx context.Context, dataID string) (string, error) {
	if runtime == nil || runtime.Config == nil {
		return "", errors.New("gateway nacos config runtime is not initialized")
	}
	value, err := runtime.Config.Load(ctx, dataID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", errors.New("gateway nacos config is empty")
	}
	return value, nil
}

func (runtime *NacosRuntime) ResolveTarget(ctx context.Context, serviceName string) (string, error) {
	if runtime == nil || runtime.Discovery == nil {
		return "", errors.New("gateway nacos discovery runtime is not initialized")
	}
	instances, err := runtime.Discovery.Resolve(ctx, serviceName)
	if err != nil {
		return "", err
	}
	if len(instances) == 0 {
		return "", errors.New("gateway nacos discovery returned no healthy instances")
	}
	instance := instances[0]
	return fmt.Sprintf("%s:%d", instance.IP, instance.Port), nil
}
