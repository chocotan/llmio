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
	timeout   time.Duration
	proxyURL  string
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
					password, _ := parsedURL.User.Password()
					if username != "" || password != "" {
						auth = &proxy.Auth{
							User:     username,
							Password: password,
						}
					}
				}
				
				socks5Dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, auth, dialer)
				if err == nil {
					// Use SOCKS5 dialer
					transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
						return socks5Dialer.Dial(network, addr)
					}
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
