package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/elazarl/goproxy"
	socks "golang.org/x/net/proxy"

	"proxygate/internal/auth"
	"proxygate/internal/proxy"
)

const (
	maxRetries           = 3
	errorRespMaxLength   = 500
	defaultListenAddress = ":8080"
)

// Options configures the server runtime.
type Options struct {
	ListenAddr string
	Verbose    bool
}

// Server wraps the goproxy server and upstream proxy pool.
type Server struct {
	httpProxy *goproxy.ProxyHttpServer
	pool      *proxy.Pool
	opts      Options
}

// New creates a new Server.
func New(pool *proxy.Pool, opts Options) *Server {
	if opts.ListenAddr == "" {
		opts.ListenAddr = defaultListenAddress
	}

	p := goproxy.NewProxyHttpServer()
	p.Verbose = opts.Verbose

	s := &Server{
		httpProxy: p,
		pool:      pool,
		opts:      opts,
	}

	p.ConnectDialWithReq = s.connectDialHandler
	return s
}

// ListenAndServe starts the HTTP proxy server.
func (s *Server) ListenAndServe() error {
	log.Printf("Starting HTTP proxy server on %s", s.opts.ListenAddr)
	return http.ListenAndServe(s.opts.ListenAddr, s.httpProxy)
}

func (s *Server) connectDialHandler(req *http.Request, network, addr string) (net.Conn, error) {
	stickyHeader := s.pool.StickyHeader()
	stickyKey := ""
	requestURI := ""
	if req != nil {
		stickyKey = req.Header.Get(stickyHeader)
		requestURI = req.RequestURI
		log.Printf("Headers: \n%s", req.Header)
	}

	selected, err := s.pool.Select(stickyKey)
	if err != nil {
		return nil, err
	}

	log.Printf("Sticky selection for %s -> %s://%s", requestURI, selected.Protocol, selected.Address)
	return s.newConnectDialToProxy(network, addr, stickyKey, selected)
}

func (s *Server) newConnectDialToProxy(network, addr, stickyKey string, chosen proxy.Proxy) (net.Conn, error) {
	current := chosen

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Selected proxy: %s://%s (attempt %d/%d)", current.Protocol, current.Address, attempt, maxRetries)

		if current.Protocol == "socks5" {
			conn, err := tryConnectSocks5Proxy(network, addr, current)
			if err == nil {
				return conn, nil
			}
			log.Printf("SOCKS5 connect failed: %v", err)
		}

		conn, err := s.connectHTTPProxy(network, addr, current)
		if err == nil {
			return conn, nil
		}

		log.Printf("HTTP connect failed: %v", err)
		s.pool.MarkFailed(current)

		next, nextErr := s.pool.Select("")
		if nextErr != nil {
			return nil, fmt.Errorf("failed to acquire replacement proxy: %w", nextErr)
		}
		s.pool.BindSticky(stickyKey, next)
		current = next
	}

	return nil, fmt.Errorf("failed to connect after %d attempts", maxRetries)
}

func (s *Server) connectHTTPProxy(network, addr string, upstream proxy.Proxy) (net.Conn, error) {
	proxyURL, err := upstream.URL()
	if err != nil {
		return nil, err
	}

	host := proxyURL.Host
	if !containsPort(host) {
		switch upstream.Protocol {
		case "https":
			host += ":443"
		default:
			host += ":80"
		}
	}

	connectReq := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: make(http.Header),
	}

	if upstream.Credentials != nil {
		auth.SetProxyAuthorization(connectReq, *upstream.Credentials)
	}

	conn, err := s.dial(network, host, connectReq)
	if err != nil {
		return nil, err
	}

	if err := connectReq.Write(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), connectReq)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, errorRespMaxLength))
		if readErr != nil {
			_ = conn.Close()
			return nil, readErr
		}
		_ = conn.Close()
		return nil, errors.New("proxy refused connection: " + string(body))
	}

	return conn, nil
}

func (s *Server) dial(network, addr string, ctxReq *http.Request) (net.Conn, error) {
	ctx := &goproxy.ProxyCtx{Req: ctxReq}

	if ctx.Dialer != nil {
		return ctx.Dialer(ctxReq.Context(), network, addr)
	}

	if s.httpProxy.Tr != nil && s.httpProxy.Tr.DialContext != nil {
		return s.httpProxy.Tr.DialContext(ctxReq.Context(), network, addr)
	}

	return net.Dial(network, addr)
}

func containsPort(host string) bool {
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return true
		}
		if host[i] == ']' {
			break
		}
	}
	return false
}

func tryConnectSocks5Proxy(network, addr string, upstream proxy.Proxy) (net.Conn, error) {
	var credentials *socks.Auth
	if upstream.Credentials != nil && upstream.Credentials.IsValid() {
		credentials = &socks.Auth{
			User:     upstream.Credentials.Username,
			Password: upstream.Credentials.Password,
		}
	}

	dialer, err := socks.SOCKS5("tcp", upstream.Address, credentials, socks.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}

	return dialer.Dial(network, addr)
}
