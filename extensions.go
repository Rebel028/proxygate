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

func connectDialHandler(req *http.Request, network string, addr string) (net.Conn, error) {
	// Sticky header to pin upstream proxy selection
	const stickyHeader = "X-Proxy-Session"
	log.Printf("Headers: \n%s", req.Header)
	log.Printf("Cookies: \n%s", req.Cookies())
	sessionKey := ""
	if req != nil {
		sessionKey = req.Header.Get(stickyHeader)
	}

	var chosen Proxy
	if sessionKey != "" {
		if v, ok := stickySessionToProxy.Load(sessionKey); ok {
			chosen = v.(Proxy)
		} else {
			chosen = getRandomProxy()
			stickySessionToProxy.Store(sessionKey, chosen)
		}
	} else {
		chosen = getRandomProxy()
	}

	log.Printf("Sticky selection for %s -> %s://%s", req.RequestURI, chosen.Protocol, chosen.Address)

	return newConnectDialToProxy(network, addr, chosen)
}

func newConnectDialToProxy(network, addr string, selectedProxy Proxy) (conn net.Conn, err error) {
	const maxRetries = 3
	p := selectedProxy
	for retry := 0; retry < maxRetries; retry++ {

		log.Printf("Selected proxy: %s://%s (attempt %d/%d)", p.Protocol, p.Address, retry+1, maxRetries)

		// Handle SOCKS5 proxies differently
		if p.Protocol == "socks5" {
			conn, err = tryConnectSocks5Proxy(network, addr, p)
			if err == nil {
				return
			}
		}

		// Handle HTTP/HTTPS proxies
		conn, err = connectHttpProxy(network, addr, p)
		if err == nil {
			return
		}

		markProxyAsFailed(p)
		p = getRandomProxy()
		continue
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %v", maxRetries, err)
}

func connectHttpProxy(network string, addr string, p Proxy) (net.Conn, error) {
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
		return nil, err
	}

	if err := connectReq.Write(c); err != nil {
		_ = c.Close()
		return nil, err
	}

	// Read response.
	// Okay to use and discard buffered reader here, because
	// TLS server will not speak until spoken to.
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, connectReq)
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, _errorRespMaxLength))
		if err != nil {
			_ = c.Close()
			return nil, err
		}
		_ = c.Close()
		err = errors.New("proxy refused connection: " + string(respBody))
		return nil, err
	}

	return c, nil
}

func tryConnectSocks5Proxy(network string, addr string, p Proxy) (net.Conn, error) {
	var auth *socks.Auth
	var err error
	if p.Auth != nil && p.Auth.Username != "" && p.Auth.Password != "" {
		auth = &socks.Auth{
			User:     p.Auth.Username,
			Password: p.Auth.Password,
		}
	}

	dialer, err := socks.SOCKS5("tcp", p.Address, auth, socks.Direct)
	if err != nil {
		err = fmt.Errorf("failed to create SOCKS5 dialer: %v", err)
		return nil, err
	}

	return dialer.Dial(network, addr)
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
