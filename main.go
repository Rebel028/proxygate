package main

import (
	"flag"
	"fmt"
	"github.com/elazarl/goproxy"
	"golang.org/x/net/proxy" // Use this for SOCKS5 proxy support
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync"
)

type Proxy struct {
	Protocol string
	Address  string
	Auth     string
}

var (
	proxyPool    []Proxy //todo: replace with sqlite
	mutex        = &sync.Mutex{}
	basicAuth    bool
	httpUsername string
	httpPassword string
	httpsMitm    bool
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
	if envMitm := os.Getenv("PROXY_MITM"); envMitm != "False" && envMitm != "false" {
		httpsMitm = true
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
	return proxyPool[rand.Intn(len(proxyPool))] //todo: rotate proxies with more background logic
}

func startHTTPProxyServer(addr string) {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	if httpsMitm {
		proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	}
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		if !checkBasicAuth(req) {
			return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusProxyAuthRequired, "Proxy Authentication Required")
		}

		selectedProxy := getRandomProxy()
		log.Printf("Using Proxy: %s\n", selectedProxy.Address)

		client, err := makeHTTPClient(selectedProxy)
		if err != nil {
			return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusBadGateway, err.Error())
		}

		// Create a new request for forwarding
		forwardReq, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
		if err != nil {
			return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusBadGateway, err.Error())
		}

		// Copy the headers
		forwardReq.Header = req.Header.Clone()

		// Forward the request using the newly created client
		resp, err := client.Do(forwardReq)
		if err != nil {
			//todo: handle bad proxy
			return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusBadGateway, err.Error())
		}
		return req, resp
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

func makeHTTPClient(pr Proxy) (*http.Client, error) {
	proxyURL, err := url.Parse(buildProxyURL(pr))
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %v", err)
	}

	if pr.Protocol == "socks5" {
		dialer, err := proxy.SOCKS5("tcp", pr.Address, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 proxy: %v", err)
		}
		return &http.Client{
			Transport: &http.Transport{
				Dial: dialer.Dial,
			},
		}, nil
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}, nil
}

func buildProxyURL(proxy Proxy) string {
	if proxy.Auth != "" {
		return fmt.Sprintf("%s://%s@%s", proxy.Protocol, proxy.Auth, proxy.Address)
	}
	return fmt.Sprintf("%s://%s", proxy.Protocol, proxy.Address)
}
