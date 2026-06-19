package bmkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"earthquake-api/internal/config"
	"earthquake-api/internal/model"
)

const (
	pathAutoGempa     = "/autogempa.json"
	pathGempaTerkini  = "/gempaterkini.json"
	pathGempaDirasakan = "/gempadirasakan.json"
)

// Client defines the interface for BMKG API operations.
type Client interface {
	FetchLatest() (*model.Earthquake, error)
	FetchM5Plus() ([]model.Earthquake, error)
	FetchFelt() ([]model.Earthquake, error)
	ShakemapURL(code string) string
}

type client struct {
	baseURL    string
	jsonPath   string
	shakemap   string
	httpClient *http.Client
	maxRetries int
	retryWait  time.Duration
}

// New creates a new BMKG API client.
func New(cfg config.BMKGConfig) Client {
	return &client{
		baseURL:  cfg.BaseURL,
		jsonPath: cfg.JSONEndpoint,
		shakemap: cfg.ShakemapURL,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
		maxRetries: cfg.MaxRetries,
		retryWait:  cfg.RetryWait,
	}
}

func (c *client) FetchLatest() (*model.Earthquake, error) {
	url := c.buildURL(pathAutoGempa)
	var resp model.AutoGempaResponse
	if err := c.getJSON(url, &resp); err != nil {
		return nil, fmt.Errorf("fetch latest: %w", err)
	}
	return &resp.Infogempa.Gempa, nil
}

func (c *client) FetchM5Plus() ([]model.Earthquake, error) {
	url := c.buildURL(pathGempaTerkini)
	var resp model.GempaListResponse
	if err := c.getJSON(url, &resp); err != nil {
		return nil, fmt.Errorf("fetch m5+: %w", err)
	}
	return resp.Infogempa.Gempa, nil
}

func (c *client) FetchFelt() ([]model.Earthquake, error) {
	url := c.buildURL(pathGempaDirasakan)
	var resp model.GempaListResponse
	if err := c.getJSON(url, &resp); err != nil {
		return nil, fmt.Errorf("fetch felt: %w", err)
	}
	return resp.Infogempa.Gempa, nil
}

func (c *client) ShakemapURL(code string) string {
	return fmt.Sprintf("%s/%s", c.shakemap, code)
}

func (c *client) buildURL(path string) string {
	return c.baseURL + c.jsonPath + path
}

// getJSON performs an HTTP GET with retries and decodes JSON response.
func (c *client) getJSON(urlStr string, target interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryWait)
		}

		req, err := http.NewRequest(http.MethodGet, urlStr, nil)
		if err != nil {
			lastErr = fmt.Errorf("create request: %w", err)
			continue
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "BMKG-Earthquake-API/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http get: %w", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = model.ErrRateLimited
			// Don't retry on rate limit — wait would just compound
			return fmt.Errorf("bmkg rate limited (429): %w", lastErr)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("unexpected status %d: %s", resp.StatusCode, truncate(string(body), 200))
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("read body: %w", err)
			continue
		}

		if err := json.Unmarshal(body, target); err != nil {
			lastErr = fmt.Errorf("json decode: %w", err)
			continue
		}
		return nil
	}

	return fmt.Errorf("%w: %v", model.ErrUpstream, lastErr)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ValidateURL checks if a shakemap code is safe to use in a URL.
func ValidateURL(code string) error {
	if code == "" {
		return fmt.Errorf("shakemap code is required")
	}
	decoded, err := url.QueryUnescape(code)
	if err != nil {
		return fmt.Errorf("invalid shakemap code: %w", err)
	}
	if decoded != code {
		return fmt.Errorf("shakemap code must not be URL-encoded")
	}
	// Only allow alphanumeric, dots, hyphens, and underscores
	for _, r := range code {
		if !isAlphaNum(r) && r != '.' && r != '-' && r != '_' && r != '/' {
			return fmt.Errorf("shakemap code contains invalid character: %c", r)
		}
	}
	return nil
}

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
