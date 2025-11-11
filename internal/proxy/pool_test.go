package proxy

import (
	"os"
	"path/filepath"
	"testing"

	"proxygate/internal/auth"
)

func TestLoadFromFileParsesEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxies.txt")

	content := `
# comment line
127.0.0.1:8080:alice:secret
http://bob:pass@example.com:3128
https://example.org:443
`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	defaultCred := &auth.Credentials{Username: "global", Password: "pw"}

	pool, err := LoadFromFile(path, Options{DefaultCredentials: defaultCred})
	if err != nil {
		t.Fatalf("LoadFromFile returned error: %v", err)
	}

	if pool.Len() != 3 {
		t.Fatalf("expected 3 proxies, got %d", pool.Len())
	}

	first := pool.proxies[0]
	if first.Protocol != "http" || first.Address != "127.0.0.1:8080" {
		t.Fatalf("unexpected first proxy: %+v", first)
	}
	if first.Credentials == nil || first.Credentials.Username != "alice" {
		t.Fatalf("expected explicit credentials for first proxy")
	}

	third := pool.proxies[2]
	if third.Protocol != "https" {
		t.Fatalf("expected https protocol, got %s", third.Protocol)
	}
	if third.Credentials == nil || third.Credentials.Username != "global" {
		t.Fatalf("expected default credentials applied to third proxy")
	}
}

func TestSelectStickyReusesProxy(t *testing.T) {
	pool := NewPool(Options{})
	pool.SetProxies([]Proxy{
		{Protocol: "http", Address: "one"},
		{Protocol: "http", Address: "two"},
	})

	first, err := pool.Select("session-1")
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}

	second, err := pool.Select("session-1")
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}

	if first != second {
		t.Fatalf("expected sticky selection to reuse proxy, got %+v vs %+v", first, second)
	}

	pool.MarkFailed(first)

	if value, ok := pool.sticky.Load("session-1"); ok && value.(Proxy) == first {
		t.Fatalf("expected sticky entry to be cleared after failure")
	}
}
