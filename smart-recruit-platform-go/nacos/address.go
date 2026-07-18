package nacos

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func ParseAddresses(raw string) ([]*url.URL, error) {
	parts := strings.Split(raw, ",")
	addresses := make([]*url.URL, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if !strings.Contains(item, "://") {
			item = "http://" + item
		}
		parsed, err := url.Parse(item)
		if err != nil {
			return nil, fmt.Errorf("parse nacos address %q: %w", part, err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, fmt.Errorf("unsupported nacos address scheme: %s", parsed.Scheme)
		}
		host := parsed.Hostname()
		if host == "" {
			return nil, fmt.Errorf("nacos address host is required: %q", part)
		}
		port := parsed.Port()
		if port == "" {
			port = "8848"
		}
		if _, err := strconv.Atoi(port); err != nil {
			return nil, fmt.Errorf("nacos address port is invalid: %q", part)
		}
		parsed.Host = net.JoinHostPort(host, port)
		parsed.Path = strings.TrimRight(parsed.Path, "/")
		addresses = append(addresses, parsed)
	}
	if len(addresses) == 0 {
		return nil, errors.New("at least one nacos address is required")
	}
	return addresses, nil
}
