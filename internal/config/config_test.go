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

func TestLoadUsesEnvVarsForAllFlags(t *testing.T) {
	t.Setenv("PROXY_LISTEN", ":9999")
	t.Setenv("PROXY_FILE", "env_proxies.txt")
	t.Setenv("PROXY_VERBOSE", "true")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ListenAddr != ":9999" {
		t.Fatalf("expected listen addr :9999 from env, got %s", cfg.ListenAddr)
	}

	if cfg.ProxyListPath != "env_proxies.txt" {
		t.Fatalf("expected proxy file env_proxies.txt from env, got %s", cfg.ProxyListPath)
	}

	if !cfg.Verbose {
		t.Fatalf("expected verbose to be true from env")
	}
}

func TestLoadFlagsOverrideEnvVars(t *testing.T) {
	t.Setenv("PROXY_LISTEN", ":9999")
	t.Setenv("PROXY_FILE", "env_proxies.txt")
	t.Setenv("PROXY_VERBOSE", "false")

	args := []string{
		"-listen", ":7777",
		"-proxy-file", "flag_proxies.txt",
		"-verbose",
	}

	cfg, err := Load(args)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ListenAddr != ":7777" {
		t.Fatalf("expected listen addr :7777 from flag, got %s", cfg.ListenAddr)
	}

	if cfg.ProxyListPath != "flag_proxies.txt" {
		t.Fatalf("expected proxy file flag_proxies.txt from flag, got %s", cfg.ProxyListPath)
	}

	if !cfg.Verbose {
		t.Fatalf("expected verbose to be true from flag override")
	}
}

func TestGetBoolEnvOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"true", "true", true},
		{"True", "True", true},
		{"TRUE", "TRUE", true},
		{"1", "1", true},
		{"yes", "yes", true},
		{"Yes", "Yes", true},
		{"on", "on", true},
		{"false", "false", false},
		{"0", "0", false},
		{"no", "no", false},
		{"off", "off", false},
		{"empty", "", false},
		{"invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PROXY_VERBOSE", tt.envValue)
			result := getBoolEnvOrDefault(envProxyVerbose, false)
			if result != tt.expected {
				t.Fatalf("expected %v for env value %q, got %v", tt.expected, tt.envValue, result)
			}
		})
	}
}
