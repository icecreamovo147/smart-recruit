package config

import (
	"path/filepath"
	"testing"
)

func TestLoadRequiresExplicitRoot(t *testing.T) {
	_, err := Load(nil, emptyEnv)
	if err == nil {
		t.Fatal("expected missing root error")
	}
}

func TestLoadAcceptsLoopbackRootFromEnvAndFlags(t *testing.T) {
	root := t.TempDir()
	cfg, err := Load([]string{"--addr", "localhost:8092"}, func(key string) (string, bool) {
		if key == "DEV_LOG_VIEWER_ROOT" {
			return root, true
		}
		return "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "localhost:8092" {
		t.Fatalf("addr = %q", cfg.Addr)
	}
	if cfg.Root != filepath.Clean(root) {
		t.Fatalf("root = %q, want %q", cfg.Root, filepath.Clean(root))
	}
}

func TestLoadRejectsNonLoopbackAddr(t *testing.T) {
	_, err := Load([]string{"--root", t.TempDir(), "--addr", "0.0.0.0:8090"}, emptyEnv)
	if err == nil {
		t.Fatal("expected non-loopback rejection")
	}
}

func emptyEnv(string) (string, bool) {
	return "", false
}
