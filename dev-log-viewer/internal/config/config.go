package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

const (
	DefaultAddr = "127.0.0.1:8090"
)

type Config struct {
	Addr string
	Root string
}

type LookupEnv func(string) (string, bool)

func Load(args []string, lookup LookupEnv) (Config, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	cfg := Config{
		Addr: envOrDefault(lookup, "DEV_LOG_VIEWER_ADDR", DefaultAddr),
		Root: envOrDefault(lookup, "DEV_LOG_VIEWER_ROOT", ""),
	}

	flags := flag.NewFlagSet("dev-log-viewer", flag.ContinueOnError)
	flags.StringVar(&cfg.Addr, "addr", cfg.Addr, "loopback HTTP address")
	flags.StringVar(&cfg.Root, "root", cfg.Root, "repository root")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	if err := validateLoopback(cfg.Addr); err != nil {
		return Config{}, err
	}
	if cfg.Root == "" {
		return Config{}, errors.New("repository root is required")
	}
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return Config{}, err
	}
	cfg.Root = filepath.Clean(root)
	return cfg, nil
}

func envOrDefault(lookup LookupEnv, key string, fallback string) string {
	if value, ok := lookup(key); ok && value != "" {
		return value
	}
	return fallback
}

func validateLoopback(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", addr, err)
	}
	if port == "" {
		return fmt.Errorf("invalid address %q: port is required", addr)
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("address %q must bind to loopback", addr)
	}
	return nil
}
