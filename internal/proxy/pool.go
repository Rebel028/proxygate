package proxy

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"proxygate/internal/auth"
)

var (
	ipPortPattern = regexp.MustCompile(`^([\d\.]+):(\d+)(?::([^:]+):([^:]+))?$`)
)

// Proxy models a single upstream proxy server configuration.
type Proxy struct {
	Protocol    string
	Address     string
	Credentials *auth.Credentials
}

// URL returns the proxy as a URL instance.
func (p Proxy) URL() (*url.URL, error) {
	if p.Protocol == "" {
		return nil, errors.New("proxy protocol is required")
	}
	if p.Address == "" {
		return nil, errors.New("proxy address is required")
	}

	var userInfo *url.Userinfo
	if p.Credentials != nil && p.Credentials.IsValid() {
		userInfo = url.UserPassword(p.Credentials.Username, p.Credentials.Password)
	}

	return &url.URL{
		Scheme: p.Protocol,
		Host:   p.Address,
		User:   userInfo,
	}, nil
}

// Pool manages upstream proxies and sticky-session mapping.
type Pool struct {
	mu              sync.RWMutex
	proxies         []Proxy
	random          *rand.Rand
	sticky          sync.Map
	defaultCred     *auth.Credentials
	stickyHeaderKey string
}

// Options configures a Pool.
type Options struct {
	DefaultCredentials *auth.Credentials
	StickyHeader       string
}

const defaultStickyHeader = "X-Proxy-Session"

// NewPool constructs a Pool with optional defaults.
func NewPool(opts Options) *Pool {
	source := rand.NewSource(time.Now().UnixNano())
	stickyKey := opts.StickyHeader
	if stickyKey == "" {
		stickyKey = defaultStickyHeader
	}
	return &Pool{
		random:          rand.New(source),
		defaultCred:     cloneCredentials(opts.DefaultCredentials),
		stickyHeaderKey: stickyKey,
	}
}

// StickyHeader returns the header key used for sticky sessions.
func (p *Pool) StickyHeader() string {
	return p.stickyHeaderKey
}

// SetProxies replaces the pool contents.
func (p *Pool) SetProxies(proxies []Proxy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.proxies = append([]Proxy(nil), proxies...)
}

// Len returns the number of proxies in the pool.
func (p *Pool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.proxies)
}

// Select returns a proxy, honoring sticky sessions when a key is supplied.
func (p *Pool) Select(stickyKey string) (Proxy, error) {
	if stickyKey != "" {
		if value, ok := p.sticky.Load(stickyKey); ok {
			if upstream, ok := value.(Proxy); ok {
				return upstream, nil
			}
		}
	}

	upstream, err := p.randomProxy()
	if err != nil {
		return Proxy{}, err
	}

	if stickyKey != "" {
		p.sticky.Store(stickyKey, upstream)
	}
	return upstream, nil
}

// BindSticky associates the sticky key with the provided proxy.
func (p *Pool) BindSticky(stickyKey string, upstream Proxy) {
	if stickyKey == "" {
		return
	}
	p.sticky.Store(stickyKey, upstream)
}

// MarkFailed logs the failure and evicts matching sticky-session entries.
func (p *Pool) MarkFailed(upstream Proxy) {
	log.Printf("Marking proxy as failed: %s://%s", upstream.Protocol, upstream.Address)

	p.sticky.Range(func(key, value any) bool {
		if v, ok := value.(Proxy); ok && proxiesEqual(v, upstream) {
			p.sticky.Delete(key)
		}
		return true
	})
}

func (p *Pool) randomProxy() (Proxy, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.proxies) == 0 {
		return Proxy{}, errors.New("proxy pool is empty")
	}

	index := p.random.Intn(len(p.proxies))
	return p.proxies[index], nil
}

// LoadFromFile constructs a pool from the provided file path.
func LoadFromFile(path string, opts Options) (*Pool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open proxy list: %w", err)
	}
	defer file.Close()

	pool := NewPool(opts)
	scanner := bufio.NewScanner(file)

	var proxies []Proxy
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		proxy, err := parseLine(line, pool.defaultCred)
		if err != nil {
			return nil, fmt.Errorf("parse proxy at line %d: %w", lineNumber, err)
		}
		proxies = append(proxies, proxy)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan proxy list: %w", err)
	}

	if len(proxies) == 0 {
		return nil, errors.New("no proxies loaded from list")
	}

	pool.SetProxies(proxies)
	return pool, nil
}

func parseLine(line string, defaultCred *auth.Credentials) (Proxy, error) {
	if proxy, ok := tryParseColonFormat(line, defaultCred); ok {
		return proxy, nil
	}
	return parseURLFormat(line, defaultCred)
}

func tryParseColonFormat(line string, defaultCred *auth.Credentials) (Proxy, bool) {
	matches := ipPortPattern.FindStringSubmatch(line)
	if matches == nil {
		return Proxy{}, false
	}

	var credentials *auth.Credentials
	switch len(matches) {
	case 5:
		credentials = &auth.Credentials{
			Username: matches[3],
			Password: matches[4],
		}
	default:
		if defaultCred != nil && defaultCred.IsValid() {
			credentials = cloneCredentials(defaultCred)
		}
	}

	return Proxy{
		Protocol:    "http",
		Address:     fmt.Sprintf("%s:%s", matches[1], matches[2]),
		Credentials: credentials,
	}, true
}

func parseURLFormat(line string, defaultCred *auth.Credentials) (Proxy, error) {
	parsedURL, err := url.Parse(line)
	if err != nil {
		return Proxy{}, fmt.Errorf("invalid proxy URL: %w", err)
	}

	protocol := strings.ToLower(parsedURL.Scheme)
	if protocol == "socks" {
		protocol = "socks5"
	}

	var credentials *auth.Credentials
	if parsedURL.User != nil {
		password, _ := parsedURL.User.Password()
		credentials = &auth.Credentials{
			Username: parsedURL.User.Username(),
			Password: password,
		}
	} else if defaultCred != nil && defaultCred.IsValid() {
		credentials = cloneCredentials(defaultCred)
	}

	if parsedURL.Host == "" {
		return Proxy{}, errors.New("proxy url missing host")
	}

	return Proxy{
		Protocol:    protocol,
		Address:     parsedURL.Host,
		Credentials: credentials,
	}, nil
}

func proxiesEqual(a, b Proxy) bool {
	return a.Protocol == b.Protocol && a.Address == b.Address && credentialsEqual(a.Credentials, b.Credentials)
}

func credentialsEqual(a, b *auth.Credentials) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return a.Username == b.Username && a.Password == b.Password
	}
}

func cloneCredentials(c *auth.Credentials) *auth.Credentials {
	if c == nil {
		return nil
	}
	if !c.IsValid() {
		return nil
	}
	clone := *c
	return &clone
}
