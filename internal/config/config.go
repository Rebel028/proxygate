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

	envProxyUser    = "PROXY_USER"
	envProxyPass    = "PROXY_PASS"
	envProxyListen  = "PROXY_LISTEN"
	envProxyFile    = "PROXY_FILE"
	envProxyVerbose = "PROXY_VERBOSE"
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
// Environment variables are used as defaults, and command-line flags override them.
func Load(args []string) (Config, error) {
	flagSet := flag.NewFlagSet("proxygate", flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)

	// Get defaults from environment variables
	listenDefault := getEnvOrDefault(envProxyListen, defaultListenAddr)
	proxyFileDefault := getEnvOrDefault(envProxyFile, defaultProxyFile)
	verboseDefault := getBoolEnvOrDefault(envProxyVerbose, false)

	var cfg Config
	flagSet.StringVar(&cfg.ListenAddr, "listen", listenDefault, "Address for the HTTP proxy server to listen on (env: PROXY_LISTEN)")
	flagSet.StringVar(&cfg.ProxyListPath, "proxy-file", proxyFileDefault, "Path to the proxy list file (env: PROXY_FILE)")

	userFlag := flagSet.String("user", "", "Username for HTTP proxy basic authentication (env: PROXY_USER)")
	passFlag := flagSet.String("pass", "", "Password for HTTP proxy basic authentication (env: PROXY_PASS)")
	flagSet.BoolVar(&cfg.Verbose, "verbose", verboseDefault, "Enable verbose logging for proxy handler (env: PROXY_VERBOSE)")

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

// getEnvOrDefault returns the environment variable value if set, otherwise returns the default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

// getBoolEnvOrDefault returns the boolean value of the environment variable if set, otherwise returns the default.
// Accepts "true", "1", "yes", "on" (case-insensitive) as true, everything else as false.
func getBoolEnvOrDefault(key string, defaultValue bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes" || value == "on"
}
