package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
)

func loadProxies(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open proxy list: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		proxy, _ := parseProxy(line)
		proxyPool = append(proxyPool, proxy)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading proxy list: %v", err)
	}
	fmt.Printf("Loaded %d proxies.\n", len(proxyPool))
}

var ipPortPattern = regexp.MustCompile(`^([\d\.]+):(\d+)(?::([^:]+):([^:]+))?$`)

func parseProxy(line string) (*Proxy, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	// Try parsing host:port:user:password format
	p, success := TryParseColonFormatted(line)
	if success {
		return p, nil
	}
	// Try parsing URL format
	return TryParseUrlFormatted(line)
}

// TryParseUrlFormatted parsing URL formatted proxy
func TryParseUrlFormatted(line string) (*Proxy, error) {
	parsedURL, err := url.Parse(line)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy format: %s", line)
	}

	protocol := parsedURL.Scheme
	if protocol == "socks" {
		protocol = "socks5"
	}

	proxy := &Proxy{
		Protocol: protocol,
		Address:  parsedURL.Host,
	}

	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		password, _ := parsedURL.User.Password()
		proxy.Auth = &Auth{
			Username: username,
			Password: password,
		}
	} else {
		if globalAuthCredentialsSupplied {
			proxy.Auth = &Auth{
				Username: httpUsername,
				Password: httpPassword,
			}
		}
	}

	return proxy, nil
}

// TryParseColonFormatted parsing host:port:user:password format
func TryParseColonFormatted(line string) (*Proxy, bool) {
	if matches := ipPortPattern.FindStringSubmatch(line); matches != nil {
		var username, password string //todo: use more elegant way to define basic auth
		if len(matches) == 5 {        // means we have host:port:user:password
			username = matches[3]
			password = matches[4]
		} else {
			// if credentials are supplied globally
			if globalAuthCredentialsSupplied {
				username = httpUsername
				password = httpPassword
			}
		}
		return &Proxy{
			Protocol: "http", // Default to HTTP for ip:port format
			Address:  fmt.Sprintf("%s:%s", matches[1], matches[2]),
			Auth: &Auth{
				Username: username,
				Password: password,
			},
		}, true
	}
	return nil, false
}

func (p Proxy) getUrl() *url.URL {
	var str string
	if p.Auth != nil && p.Auth.Username != "" && p.Auth.Password != "" {
		str = fmt.Sprintf("%s://%s:%s@%s", p.Protocol, p.Auth.Username, p.Auth.Password, p.Address)
	} else {
		str = fmt.Sprintf("%s://%s", p.Protocol, p.Address)
	}
	proxyUrl, _ := url.Parse(str)
	return proxyUrl
}
