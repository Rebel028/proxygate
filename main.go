package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"

	"github.com/elazarl/goproxy"
)

type Proxy struct {
	Protocol string
	Address  string
	Auth     *Auth
}

type Auth struct {
	Username string
	Password string
}

var (
	proxy                   *goproxy.ProxyHttpServer
	proxyPool               []*Proxy //todo: replace with sqlite
	mutex                   = &sync.Mutex{}
	proxyServerAuthRequired bool
	httpUsername            string
	httpPassword            string

	// sticky session support
	// maps header value -> Proxy
	stickySessionToProxy sync.Map
)

func init() {
	flag.StringVar(&httpUsername, "user", "", "Username for HTTP proxy basic authentication")
	flag.StringVar(&httpPassword, "pass", "", "Password for HTTP proxy basic authentication")
}

func main() {
	flag.Parse()

	loadEnvCredentials()
	validateCredentials()

	loadProxies("proxy_list.txt")
	startHTTPProxyServer(":8080")
}

func loadEnvCredentials() {
	if envUser := os.Getenv("PROXY_USER"); httpUsername == "" && envUser != "" {
		httpUsername = envUser
	}
	if envPass := os.Getenv("PROXY_PASS"); httpPassword == "" && envPass != "" {
		httpPassword = envPass
	}
}

func validateCredentials() {
	if httpUsername == "" || httpPassword == "" {
		proxyServerAuthRequired = false
	} else {
		proxyServerAuthRequired = true
	}
}

func getRandomProxy() Proxy {
	mutex.Lock()
	defer mutex.Unlock()

	if len(proxyPool) == 0 {
		log.Println("Warning: Proxy pool is empty")
		log.Fatal("No proxies available")
	}

	return *proxyPool[rand.Intn(len(proxyPool))] //todo: rotate proxies with more background logic
}

func startHTTPProxyServer(addr string) {
	proxy = goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	// Configure to handle CONNECT for HTTPS without MITM
	proxy.ConnectDialWithReq = connectDialHandler

	log.Printf("Starting HTTP proxy server on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, proxy))
}
