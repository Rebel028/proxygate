package config

import "testing"

func TestLoadUsesFlags(t *testing.T) {
	args := []string{
		"-user", "alice",
		"-pass", "secret",
		"-listen", ":9090",
		"-proxy-file", "custom.txt",
		"-verbose",
	}

	cfg, err := Load(args)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ListenAddr != ":9090" {
		t.Fatalf("expected listen addr :9090, got %s", cfg.ListenAddr)
	}

	if cfg.ProxyListPath != "custom.txt" {
		t.Fatalf("expected proxy list custom.txt, got %s", cfg.ProxyListPath)
	}

	if !cfg.RequireAuth {
		t.Fatalf("expected RequireAuth to be true")
	}

	if cfg.ServerCredentials.Username != "alice" || cfg.ServerCredentials.Password != "secret" {
		t.Fatalf("unexpected credentials: %+v", cfg.ServerCredentials)
	}

	if !cfg.Verbose {
		t.Fatalf("expected verbose flag to be true")
	}
}

func TestLoadFallsBackToEnv(t *testing.T) {
	t.Setenv("PROXY_USER", "bob")
	t.Setenv("PROXY_PASS", "hunter2")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if !cfg.RequireAuth {
		t.Fatalf("expected RequireAuth when env credentials present")
	}

	if cfg.ServerCredentials.Username != "bob" || cfg.ServerCredentials.Password != "hunter2" {
		t.Fatalf("unexpected credentials: %+v", cfg.ServerCredentials)
	}
}

func TestLoadReturnsErrorForIncompleteCredentials(t *testing.T) {
	t.Setenv("PROXY_USER", "")
	t.Setenv("PROXY_PASS", "")

	args := []string{"-user", "onlyuser"}
	if _, err := Load(args); err == nil {
		t.Fatalf("expected error when only username provided")
	}

	t.Setenv("PROXY_USER", "")
	t.Setenv("PROXY_PASS", "onlypass")

	if _, err := Load(nil); err == nil {
		t.Fatalf("expected error when only password provided via env")
	}
}
