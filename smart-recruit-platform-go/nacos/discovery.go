package nacos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Discovery interface {
	Register(ctx context.Context, instance Instance) error
	Deregister(ctx context.Context, instance Instance) error
	Resolve(ctx context.Context, serviceName string) ([]Instance, error)
}

type DiscoveryOptions struct {
	Client         *HTTPClient
	Addresses      string
	Namespace      string
	Group          string
	Env            string
	StaticFallback map[string][]Instance
	AllowFallback  bool
	Username       string
	Password       string
}

type Instance struct {
	ServiceName string
	IP          string
	Port        int
	Healthy     bool
	Metadata    map[string]string
}

type HTTPDiscovery struct {
	client    *HTTPClient
	namespace string
	group     string
}

type StaticDiscovery struct {
	instances map[string][]Instance
}

func NewDiscovery(options DiscoveryOptions) (Discovery, error) {
	group := defaultGroup(options.Group)
	if options.Client != nil {
		return HTTPDiscovery{client: options.Client, namespace: options.Namespace, group: group}, nil
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
		return HTTPDiscovery{client: client, namespace: options.Namespace, group: group}, nil
	}
	if options.AllowFallback && isLocalEnv(options.Env) {
		return StaticDiscovery{instances: cloneInstances(options.StaticFallback)}, nil
	}
	return nil, errors.New("nacos discovery requires NACOS_ADDR outside local fallback")
}

func (d HTTPDiscovery) Register(ctx context.Context, instance Instance) error {
	if err := instance.Validate(); err != nil {
		return err
	}
	values := d.instanceValues(instance)
	_, err := d.client.post(ctx, "/nacos/v1/ns/instance", values)
	return err
}

func (d HTTPDiscovery) Deregister(ctx context.Context, instance Instance) error {
	if err := instance.Validate(); err != nil {
		return err
	}
	values := d.instanceValues(instance)
	_, err := d.client.delete(ctx, "/nacos/v1/ns/instance", values)
	return err
}

func (d HTTPDiscovery) Resolve(ctx context.Context, serviceName string) ([]Instance, error) {
	if strings.TrimSpace(serviceName) == "" {
		return nil, errors.New("service name is required")
	}
	values := url.Values{}
	values.Set("serviceName", strings.TrimSpace(serviceName))
	values.Set("groupName", defaultGroup(d.group))
	if strings.TrimSpace(d.namespace) != "" {
		values.Set("namespaceId", strings.TrimSpace(d.namespace))
	}
	body, err := d.client.get(ctx, "/nacos/v1/ns/instance/list", values)
	if err != nil {
		return nil, err
	}
	var response struct {
		Hosts []struct {
			IP       string            `json:"ip"`
			Port     int               `json:"port"`
			Healthy  bool              `json:"healthy"`
			Metadata map[string]string `json:"metadata"`
		} `json:"hosts"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode nacos discovery response: %w", err)
	}
	instances := make([]Instance, 0, len(response.Hosts))
	for _, host := range response.Hosts {
		if !host.Healthy {
			continue
		}
		instances = append(instances, Instance{
			ServiceName: strings.TrimSpace(serviceName),
			IP:          host.IP,
			Port:        host.Port,
			Healthy:     host.Healthy,
			Metadata:    host.Metadata,
		})
	}
	return instances, nil
}

func (d HTTPDiscovery) instanceValues(instance Instance) url.Values {
	values := url.Values{}
	values.Set("serviceName", strings.TrimSpace(instance.ServiceName))
	values.Set("ip", strings.TrimSpace(instance.IP))
	values.Set("port", strconv.Itoa(instance.Port))
	values.Set("groupName", defaultGroup(d.group))
	if strings.TrimSpace(d.namespace) != "" {
		values.Set("namespaceId", strings.TrimSpace(d.namespace))
	}
	return values
}

func (d StaticDiscovery) Register(context.Context, Instance) error {
	return nil
}

func (d StaticDiscovery) Deregister(context.Context, Instance) error {
	return nil
}

func (d StaticDiscovery) Resolve(_ context.Context, serviceName string) ([]Instance, error) {
	instances := d.instances[strings.TrimSpace(serviceName)]
	if len(instances) == 0 {
		return nil, errors.New("static discovery fallback has no instances")
	}
	return append([]Instance(nil), instances...), nil
}

func (instance Instance) Validate() error {
	if strings.TrimSpace(instance.ServiceName) == "" {
		return errors.New("service name is required")
	}
	if strings.TrimSpace(instance.IP) == "" {
		return errors.New("instance ip is required")
	}
	if instance.Port <= 0 || instance.Port > 65535 {
		return errors.New("instance port is invalid")
	}
	return nil
}

func cloneInstances(values map[string][]Instance) map[string][]Instance {
	cloned := make(map[string][]Instance, len(values))
	for key, instances := range values {
		cloned[key] = append([]Instance(nil), instances...)
	}
	return cloned
}
