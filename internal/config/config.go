package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"proxygate/internal/auth"
)

const (
	defaultListenAddr = ":8080"
	defaultProxyFile  = "proxy_list.txt"

	envProxyUser = "PROXY_USER"
	envProxyPass = "PROXY_PASS"
)

// Config captures runtime configuration for the proxy server.
type Config struct {
	ListenAddr        string
	ProxyListPath     string
	Verbose           bool
	RequireAuth       bool
	ServerCredentials auth.Credentials
}

// Load parses configuration from command-line flags and environment variables.
func Load(args []string) (Config, error) {
	flagSet := flag.NewFlagSet("proxygate", flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)

	var cfg Config
	flagSet.StringVar(&cfg.ListenAddr, "listen", defaultListenAddr, "Address for the HTTP proxy server to listen on")
	flagSet.StringVar(&cfg.ProxyListPath, "proxy-file", defaultProxyFile, "Path to the proxy list file")

	userFlag := flagSet.String("user", "", "Username for HTTP proxy basic authentication")
	passFlag := flagSet.String("pass", "", "Password for HTTP proxy basic authentication")
	flagSet.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose logging for proxy handler")

	if err := flagSet.Parse(args); err != nil {
		return Config{}, err
	}

	cred, requireAuth, err := resolveCredentials(*userFlag, *passFlag)
	if err != nil {
		return Config{}, err
	}

	cfg.ServerCredentials = cred
	cfg.RequireAuth = requireAuth

	if cfg.ProxyListPath == "" {
		return Config{}, errors.New("proxy list path cannot be empty")
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultListenAddr
	}

	return cfg, nil
}

func resolveCredentials(user, pass string) (auth.Credentials, bool, error) {
	if user == "" {
		user = strings.TrimSpace(os.Getenv(envProxyUser))
	}
	if pass == "" {
		pass = strings.TrimSpace(os.Getenv(envProxyPass))
	}

	credentials := auth.Credentials{
		Username: user,
		Password: pass,
	}

	switch {
	case credentials.Username == "" && credentials.Password == "":
		return auth.Credentials{}, false, nil
	case credentials.Username == "" || credentials.Password == "":
		return auth.Credentials{}, false, fmt.Errorf("incomplete credentials: both username and password are required")
	default:
		return credentials, true, nil
	}
}
