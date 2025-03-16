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

	if matches := ipPortPattern.FindStringSubmatch(line); matches != nil {
		return &Proxy{
			Protocol: "http", // Default to HTTP for ip:port format
			Address:  fmt.Sprintf("%s:%s", matches[1], matches[2]),
			Auth: &Auth{
				Username: matches[3],
				Password: matches[4],
			},
		}, nil
	}

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
	}

	return proxy, nil
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
