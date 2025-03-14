package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
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
		proxy := parseProxy(line)
		proxyPool = append(proxyPool, proxy)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading proxy list: %v", err)
	}
	fmt.Printf("Loaded %d proxies.\n", len(proxyPool))
}

func parseProxy(line string) Proxy {
	protocol := "http" // default protocol
	address := line
	auth := ""

	parts := strings.Split(line, "://")
	if len(parts) == 2 {
		protocolParts := parts[0]
		address = parts[1]
		if strings.HasPrefix(protocolParts, "socks") {
			protocol = "socks5"
		} else {
			protocol = protocolParts
		}
	}

	authParts := strings.Split(address, "@")
	if len(authParts) == 2 {
		auth = authParts[0]
		address = authParts[1]
	}

	return Proxy{Protocol: protocol, Address: address, Auth: auth}
}
