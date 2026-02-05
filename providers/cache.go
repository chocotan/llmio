package providers

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

const (
	// DefaultModelsTimeout is the default timeout for fetching provider models
	DefaultModelsTimeout = 30 * time.Second
)

type clientCacheKey struct {
	timeout  time.Duration
	proxyURL string
}

type clientCache struct {
	mu      sync.RWMutex
	clients map[clientCacheKey]*http.Client
}

var cache = &clientCache{
	clients: make(map[clientCacheKey]*http.Client),
}

var dialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

// contextAwareDialer wraps a proxy.Dialer to respect context cancellation and deadlines
type contextAwareDialer struct {
	dialer     proxy.Dialer
	baseDialer *net.Dialer
}

func (d *contextAwareDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// Check if context is already done
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Use a channel to handle the dial operation with context
	type result struct {
		conn net.Conn
		err  error
	}
	resultChan := make(chan result, 1)

	go func() {
		conn, err := d.dialer.Dial(network, addr)
		resultChan <- result{conn: conn, err: err}
	}()

	select {
	case <-ctx.Done():
		// Context cancelled, return error
		return nil, ctx.Err()
	case r := <-resultChan:
		return r.conn, r.err
	}
}

// GetClient returns an http.Client with the specified responseHeaderTimeout.
// If a client with the same timeout already exists, it returns the cached one.
// Otherwise, it creates a new client and caches it.
func GetClient(responseHeaderTimeout time.Duration) *http.Client {
	return GetClientWithProxy(responseHeaderTimeout, "")
}

// GetClientWithProxy returns an http.Client with the specified responseHeaderTimeout and proxy.
// Supports HTTP and SOCKS5 proxies.
// proxyURL examples: "http://proxy:8080", "socks5://proxy:1080", "socks5://user:pass@proxy:1080"
func GetClientWithProxy(responseHeaderTimeout time.Duration, proxyURL string) *http.Client {
	key := clientCacheKey{
		timeout:  responseHeaderTimeout,
		proxyURL: proxyURL,
	}

	cache.mu.RLock()
	if client, exists := cache.clients[key]; exists {
		cache.mu.RUnlock()
		return client
	}
	cache.mu.RUnlock()

	cache.mu.Lock()
	defer cache.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := cache.clients[key]; exists {
		return client
	}

	transport := &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: responseHeaderTimeout,
	}

	// Configure proxy if provided
	if proxyURL != "" {
		parsedURL, err := url.Parse(proxyURL)
		if err == nil {
			switch parsedURL.Scheme {
			case "http", "https":
				// HTTP proxy
				transport.Proxy = http.ProxyURL(parsedURL)
				transport.DialContext = dialer.DialContext
			case "socks5":
				// SOCKS5 proxy
				var auth *proxy.Auth
				if parsedURL.User != nil {
					username := parsedURL.User.Username()
					password, hasPassword := parsedURL.User.Password()
					// Only create auth if we have actual credentials
					if username != "" && hasPassword {
						auth = &proxy.Auth{
							User:     username,
							Password: password,
						}
					}
				}

				socks5Dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, auth, dialer)
				if err == nil {
					// Use SOCKS5 dialer
					// Note: SOCKS5 dialer doesn't support context, so we wrap it
					contextDialer := &contextAwareDialer{dialer: socks5Dialer, baseDialer: dialer}
					transport.DialContext = contextDialer.DialContext
				} else {
					// Fall back to no proxy on error
					transport.Proxy = http.ProxyFromEnvironment
					transport.DialContext = dialer.DialContext
				}
			default:
				// Unknown scheme, fall back to environment proxy
				transport.Proxy = http.ProxyFromEnvironment
				transport.DialContext = dialer.DialContext
			}
		} else {
			// Parse error, fall back to environment proxy
			transport.Proxy = http.ProxyFromEnvironment
			transport.DialContext = dialer.DialContext
		}
	} else {
		// No proxy specified, use environment proxy
		transport.Proxy = http.ProxyFromEnvironment
		transport.DialContext = dialer.DialContext
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   0, // No overall timeout, let ResponseHeaderTimeout control header timing
	}

	cache.clients[key] = client
	return client
}
