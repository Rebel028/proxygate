package main

import (
	"encoding/base64"
	"flag"
	"fmt"
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
	proxyPool    []*Proxy //todo: replace with sqlite
	mutex        = &sync.Mutex{}
	basicAuth    bool
	httpUsername string
	httpPassword string
	handler      ConnectHandler
)

type ConnectHandler struct {
}

func (h ConnectHandler) HandleConnect(req string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	log.Printf("CONNECT request to: %s", req)
	return goproxy.OkConnect, req
}

const _proxyAuthHeader = "Proxy-Authorization"

func SetBasicAuth(username, password string, req *http.Request) {
	req.Header.Set(_proxyAuthHeader, "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
}

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
		basicAuth = false
	} else {
		basicAuth = true
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

// Add a new function to handle proxy failures
func markProxyAsFailed(p Proxy) {
	// In the future, this could update a database or remove the proxy from the pool
	log.Printf("Marking proxy as failed: %s://%s", p.Protocol, p.Address)
	// For now, we just log it, but this could be expanded to track failures and remove bad proxies
}

func startHTTPProxyServer(addr string) {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	// Configure to handle CONNECT for HTTPS without MITM
	proxy.OnRequest().HandleConnect(handler)
	proxy.ConnectDial = newConnectDialToRandomProxy(proxy)

	// Handle HTTP requests
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		if !checkBasicAuth(req) {
			return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusProxyAuthRequired, "Proxy Authentication Required")
		}

		log.Printf("Proxying request to: %s %s", req.Method, req.URL)

		return req, nil
	})

	// Add error handling for responses
	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp == nil {
			log.Printf("No response received for request to: %s", ctx.Req.URL)
			return nil
		}

		if resp.StatusCode >= 500 {
			log.Printf("Received error status %d for request to: %s", resp.StatusCode, ctx.Req.URL)
		}

		return resp
	})

	log.Printf("Starting HTTP proxy server on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, proxy))
}

func checkBasicAuth(req *http.Request) bool {
	if basicAuth {
		username, password, ok := req.BasicAuth()
		return ok && username == httpUsername && password == httpPassword
	}
	return true
}

func buildProxyURL(proxy Proxy) string {
	if proxy.Auth != nil {
		return fmt.Sprintf("%s://%s@%s", proxy.Protocol, proxy.Auth, proxy.Address)
	}
	return fmt.Sprintf("%s://%s", proxy.Protocol, proxy.Address)
}
