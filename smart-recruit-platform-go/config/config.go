package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"smart-recruit-platform-go/i18n"
)

type Bootstrap struct {
	ServiceName       string
	ServiceEnv        string
	ServiceVersion    string
	GRPCAddr          string
	MetricsAddr       string
	NacosAddr         string
	NacosNamespace    string
	NacosGroup        string
	NacosUsername     string
	NacosPassword     string
	MySQLDSN          string
	RedisAddr         string
	RabbitMQURL       string
	GRPCInternalToken string
	OTLPEndpoint      string
	StaticFallback    bool
	RequestTimeout    time.Duration
	AppLocale         i18n.Locale
}

type LookupFunc func(string) string

func Load() (Bootstrap, error) {
	return LoadWithLookup(os.Getenv)
}

func LoadWithLookup(lookup LookupFunc) (Bootstrap, error) {
	if lookup == nil {
		return Bootstrap{}, errors.New("config lookup is nil")
	}
	appLocale, err := i18n.ParseLocale(lookup(i18n.EnvLocale))
	if err != nil {
		return Bootstrap{}, err
	}
	if err := i18n.Configure(string(appLocale)); err != nil {
		return Bootstrap{}, err
	}
	cfg := Bootstrap{
		ServiceName:       strings.TrimSpace(lookup("SERVICE_NAME")),
		ServiceEnv:        defaultString(lookup("SERVICE_ENV"), "local"),
		ServiceVersion:    defaultString(lookup("SERVICE_VERSION"), "dev"),
		GRPCAddr:          defaultString(lookup("GRPC_ADDR"), ":0"),
		MetricsAddr:       strings.TrimSpace(lookup("METRICS_ADDR")),
		NacosAddr:         strings.TrimSpace(lookup("NACOS_ADDR")),
		NacosNamespace:    strings.TrimSpace(lookup("NACOS_NAMESPACE")),
		NacosGroup:        defaultString(lookup("NACOS_GROUP"), "DEFAULT_GROUP"),
		NacosUsername:     strings.TrimSpace(lookup("NACOS_USERNAME")),
		NacosPassword:     strings.TrimSpace(lookup("NACOS_PASSWORD")),
		MySQLDSN:          strings.TrimSpace(lookup("MYSQL_DSN")),
		RedisAddr:         strings.TrimSpace(lookup("REDIS_ADDR")),
		RabbitMQURL:       strings.TrimSpace(lookup("RABBITMQ_URL")),
		GRPCInternalToken: strings.TrimSpace(lookup("GRPC_INTERNAL_TOKEN")),
		OTLPEndpoint:      strings.TrimSpace(lookup("OTEL_EXPORTER_OTLP_ENDPOINT")),
		StaticFallback:    parseBool(lookup("STATIC_FALLBACK")),
		RequestTimeout:    defaultDuration(lookup("REQUEST_TIMEOUT"), 10*time.Second),
		AppLocale:         appLocale,
	}
	if err := cfg.Validate(); err != nil {
		return Bootstrap{}, err
	}
	return cfg, nil
}

func (cfg Bootstrap) Validate() error {
	var missing []string
	if strings.TrimSpace(cfg.ServiceName) == "" {
		missing = append(missing, "SERVICE_NAME")
	}
	if strings.TrimSpace(cfg.ServiceEnv) == "" {
		missing = append(missing, "SERVICE_ENV")
	}
	if strings.TrimSpace(cfg.ServiceVersion) == "" {
		missing = append(missing, "SERVICE_VERSION")
	}
	if strings.TrimSpace(cfg.GRPCAddr) == "" {
		missing = append(missing, "GRPC_ADDR")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required bootstrap config: %s", strings.Join(missing, ", "))
	}
	if cfg.RequestTimeout <= 0 {
		return errors.New("REQUEST_TIMEOUT must be positive")
	}
	return nil
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func parseBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

func defaultDuration(value string, fallback time.Duration) time.Duration {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}
