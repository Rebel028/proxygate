package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/elazarl/goproxy"
	socks "golang.org/x/net/proxy"
)

var _errorRespMaxLength int64 = 500

const _proxyAuthHeader = "Proxy-Authorization"

func SetBasicAuth(username, password string, req *http.Request) {
	req.Header.Set(_proxyAuthHeader, "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
}

func newConnectDialToRandomProxy(proxy *goproxy.ProxyHttpServer) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		const maxRetries = 3
		var lastErr error

		for retry := 0; retry < maxRetries; retry++ {
			p := getRandomProxy()
			log.Printf("Selected proxy: %s://%s (attempt %d/%d)", p.Protocol, p.Address, retry+1, maxRetries)

			// Handle SOCKS5 proxies differently
			if p.Protocol == "socks5" {
				var auth *socks.Auth
				if p.Auth != nil && p.Auth.Username != "" && p.Auth.Password != "" {
					auth = &socks.Auth{
						User:     p.Auth.Username,
						Password: p.Auth.Password,
					}
				}

				dialer, err := socks.SOCKS5("tcp", p.Address, auth, socks.Direct)
				if err != nil {
					lastErr = fmt.Errorf("failed to create SOCKS5 dialer: %v", err)
					markProxyAsFailed(p)
					continue
				}

				conn, err := dialer.Dial(network, addr)
				if err != nil {
					lastErr = err
					markProxyAsFailed(p)
					continue
				}
				return conn, nil
			}

			// Handle HTTP/HTTPS proxies
			proxyUrl := p.getUrl()
			if !strings.ContainsRune(proxyUrl.Host, ':') {
				if p.Protocol == "https" {
					proxyUrl.Host += ":443"
				} else {
					proxyUrl.Host += ":80"
				}
			}

			connectReq := &http.Request{
				Method: http.MethodConnect,
				URL:    &url.URL{Opaque: addr},
				Host:   addr,
				Header: make(http.Header),
			}

			if p.Auth != nil && p.Auth.Username != "" {
				SetBasicAuth(p.Auth.Username, p.Auth.Password, connectReq)
			}

			c, err := dial(proxy, &goproxy.ProxyCtx{Req: connectReq}, network, proxyUrl.Host)
			if err != nil {
				lastErr = err
				markProxyAsFailed(p)
				continue
			}

			if err := connectReq.Write(c); err != nil {
				c.Close()
				lastErr = err
				markProxyAsFailed(p)
				continue
			}

			// Read response.
			// Okay to use and discard buffered reader here, because
			// TLS server will not speak until spoken to.
			br := bufio.NewReader(c)
			resp, err := http.ReadResponse(br, connectReq)
			if err != nil {
				c.Close()
				lastErr = err
				markProxyAsFailed(p)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				respBody, err := io.ReadAll(io.LimitReader(resp.Body, _errorRespMaxLength))
				if err != nil {
					c.Close()
					lastErr = err
					markProxyAsFailed(p)
					continue
				}
				c.Close()
				lastErr = errors.New("proxy refused connection: " + string(respBody))
				markProxyAsFailed(p)
				continue
			}

			return c, nil
		}

		return nil, fmt.Errorf("failed to connect after %d attempts: %v", maxRetries, lastErr)
	}
}

func dial(proxy *goproxy.ProxyHttpServer, ctx *goproxy.ProxyCtx, network, addr string) (c net.Conn, err error) {
	if ctx.Dialer != nil {
		return ctx.Dialer(ctx.Req.Context(), network, addr)
	}

	if proxy.Tr != nil && proxy.Tr.DialContext != nil {
		return proxy.Tr.DialContext(ctx.Req.Context(), network, addr)
	}

	// if the user didn't specify any dialer, we just use the default one,
	// provided by net package
	return net.Dial(network, addr)
}

// Add a new function to handle proxy failures
func markProxyAsFailed(p Proxy) {
	// In the future, this could update a database or remove the proxy from the pool
	log.Printf("Marking proxy as failed: %s://%s", p.Protocol, p.Address)
	// For now, we just log it, but this could be expanded to track failures and remove bad proxies
}
