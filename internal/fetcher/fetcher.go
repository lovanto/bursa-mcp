// Package fetcher performs HTTP GETs against idx.co.id from behind its
// Cloudflare WAF. Phase 1 spike established that the WAF blocks plain net/http
// and Chrome-like TLS fingerprints, but a Firefox TLS profile from
// bogdanfinn/tls-client passes consistently. This package encodes that finding
// plus polite rate limiting and Cloudflare-challenge detection.
package fetcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// firefoxUserAgent must stay consistent with the Firefox TLS profile below;
// a Chrome UA over a Firefox fingerprint is itself a bot signal.
const firefoxUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0"

// ErrChallenge is returned when IDX responds with a Cloudflare challenge page
// (HTML) instead of the expected payload. Callers must not cache this.
var ErrChallenge = errors.New("cloudflare challenge / WAF block")

// Config tunes fetcher behaviour. Zero values fall back to conservative
// defaults suitable for a single-user MCP server.
type Config struct {
	// MinInterval is the minimum wall-clock gap between successive requests to
	// idx.co.id. Phase 1 recommends >=15s to stay polite; default 15s.
	MinInterval time.Duration
	// MaxRetries is how many times a request is retried on transient failure
	// (403 challenge, 5xx, network error) before giving up. Default 3.
	MaxRetries int
	// Timeout is the per-request timeout in seconds. Default 30.
	TimeoutSeconds int
}

func (c Config) withDefaults() Config {
	if c.MinInterval <= 0 {
		c.MinInterval = 15 * time.Second
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 30
	}
	return c
}

// Fetcher is the interface the rest of the app depends on, so tests can inject
// a fake instead of hitting the network.
type Fetcher interface {
	Get(ctx context.Context, url string) ([]byte, error)
}

// Client is the production Fetcher backed by tls-client.
type Client struct {
	cfg     Config
	http    tls_client.HttpClient
	mu      sync.Mutex // serialises requests and guards lastReq
	lastReq time.Time
}

// New builds a Client with a Firefox TLS profile. It returns an error only if
// the underlying TLS client cannot be constructed.
func New(cfg Config) (*Client, error) {
	cfg = cfg.withDefaults()
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(cfg.TimeoutSeconds),
		tls_client.WithClientProfile(profiles.Firefox_120),
	}
	hc, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, fmt.Errorf("build tls client: %w", err)
	}
	return &Client{cfg: cfg, http: hc}, nil
}

// Get fetches url, enforcing the rate limit and retrying transient failures.
// It returns ErrChallenge (wrapped) if every attempt hits the WAF.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with the rate-limit floor: 1x, 2x, 4x ...
			backoff := c.cfg.MinInterval << (attempt - 1)
			if err := sleep(ctx, backoff); err != nil {
				return nil, err
			}
		}
		body, err := c.doOnce(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		// Only retry transient conditions; a context cancellation is terminal.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("get %s failed after %d attempts: %w", url, c.cfg.MaxRetries+1, lastErr)
}

// doOnce performs a single rate-limited request and classifies the response.
func (c *Client) doOnce(ctx context.Context, url string) ([]byte, error) {
	if err := c.waitTurn(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", firefoxUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Referer", "https://www.idx.co.id/")
	req.Header.Set("Origin", "https://www.idx.co.id")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Cloudflare block: 403, or a 200 that is actually an HTML challenge page.
	if resp.StatusCode == http.StatusForbidden || looksLikeChallenge(resp.Header.Get("Content-Type"), body) {
		return nil, fmt.Errorf("status %d: %w", resp.StatusCode, ErrChallenge)
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return body, nil
}

// waitTurn blocks until MinInterval has elapsed since the previous request.
// The mutex also serialises concurrent callers so we never burst IDX.
func (c *Client) waitTurn(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.lastReq.IsZero() {
		wait := c.cfg.MinInterval - time.Since(c.lastReq)
		if wait > 0 {
			if err := sleep(ctx, wait); err != nil {
				return err
			}
		}
	}
	c.lastReq = time.Now()
	return nil
}

// looksLikeChallenge reports whether a 200 response is really an HTML page
// (Cloudflare interstitial) rather than the JSON/binary payload we expect.
func looksLikeChallenge(contentType string, body []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		return true
	}
	head := strings.ToLower(strings.TrimSpace(string(body[:min(len(body), 512)])))
	return strings.HasPrefix(head, "<!doctype html") ||
		strings.HasPrefix(head, "<html") ||
		strings.Contains(head, "just a moment") // Cloudflare Turnstile title
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
