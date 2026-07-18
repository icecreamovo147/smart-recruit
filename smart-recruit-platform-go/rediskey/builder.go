// Package rediskey builds service-prefixed Redis keys for the shared Redis instance.
package rediskey

import (
	"fmt"
	"strings"
)

const defaultSeparator = ":"

// Builder creates Redis keys and channels under one service prefix.
type Builder struct {
	service   string
	separator string
}

// NewBuilder returns a key builder for one service. Service names are required
// because Redis is shared by all extracted services.
func NewBuilder(service string, options ...Option) (Builder, error) {
	service = normalizePart(service)
	if service == "" {
		return Builder{}, fmt.Errorf("redis key service prefix is required")
	}
	b := Builder{service: service, separator: defaultSeparator}
	for _, option := range options {
		option(&b)
	}
	if strings.TrimSpace(b.separator) == "" {
		return Builder{}, fmt.Errorf("redis key separator is required")
	}
	return b, nil
}

// MustBuilder returns a Builder or panics. It is intended for package-level
// service key builders where invalid configuration should fail fast.
func MustBuilder(service string, options ...Option) Builder {
	b, err := NewBuilder(service, options...)
	if err != nil {
		panic(err)
	}
	return b
}

// Key builds a Redis data key as service:part[:part...].
func (b Builder) Key(parts ...string) string {
	return b.join(parts...)
}

// Channel builds a Redis Pub/Sub channel under the same service prefix.
func (b Builder) Channel(parts ...string) string {
	return b.join(parts...)
}

// Prefix returns the enforced service prefix including the separator.
func (b Builder) Prefix() string {
	return b.service + b.separator
}

// HasPrefix reports whether key belongs to this builder's service prefix.
func (b Builder) HasPrefix(key string) bool {
	return strings.HasPrefix(key, b.Prefix())
}

func (b Builder) join(parts ...string) string {
	if strings.TrimSpace(b.service) == "" {
		panic("redis key service prefix is required")
	}
	clean := make([]string, 0, len(parts)+1)
	clean = append(clean, b.service)
	for _, part := range parts {
		part = normalizePart(part)
		if part == "" {
			panic("redis key part is required")
		}
		clean = append(clean, part)
	}
	return strings.Join(clean, b.separator)
}

func normalizePart(part string) string {
	return strings.Trim(strings.TrimSpace(part), ":")
}

// Option customizes a Builder.
type Option func(*Builder)

// WithSeparator changes the separator between key parts.
func WithSeparator(separator string) Option {
	return func(b *Builder) {
		b.separator = separator
	}
}
