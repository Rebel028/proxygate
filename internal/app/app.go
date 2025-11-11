package app

import (
	"context"
	"fmt"
	"log"

	"proxygate/internal/auth"
	"proxygate/internal/config"
	"proxygate/internal/proxy"
	"proxygate/internal/server"
)

// Run is the main entrypoint used by CLI binaries.
func Run(_ context.Context, args []string) error {
	cfg, err := config.Load(args)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var defaultCred *auth.Credentials
	if cfg.RequireAuth {
		cred := cfg.ServerCredentials
		defaultCred = &cred
		log.Printf("HTTP proxy authentication enabled for user %s", cred.Username)
	} else {
		log.Printf("HTTP proxy authentication disabled")
	}

	pool, err := proxy.LoadFromFile(cfg.ProxyListPath, proxy.Options{
		DefaultCredentials: defaultCred,
	})
	if err != nil {
		return fmt.Errorf("load proxies: %w", err)
	}

	log.Printf("Loaded %d proxies from %s", pool.Len(), cfg.ProxyListPath)

	srv := server.New(pool, server.Options{
		ListenAddr: cfg.ListenAddr,
		Verbose:    cfg.Verbose,
	})

	if err := srv.ListenAndServe(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	return nil
}
