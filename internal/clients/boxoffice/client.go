package boxoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// Client wraps the external Box Office API.
type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
	logger  *slog.Logger
}

// Response represents the API response structure.
type Response struct {
	Title       string  `json:"title"`
	Distributor string  `json:"distributor"`
	ReleaseDate string  `json:"releaseDate"`
	Budget      int64   `json:"budget"`
	Revenue     Revenue `json:"revenue"`
	MPARating   string  `json:"mpaRating"`
}

// Revenue represents box office revenue data.
type Revenue struct {
	Worldwide         int64 `json:"worldwide"`
	OpeningWeekendUSA int64 `json:"openingWeekendUSA"`
}

// NewClient creates a Box Office API client with retries.
func NewClient(baseURL, apiKey string, logger *slog.Logger) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 2
	retryClient.RetryWaitMin = 100 * time.Millisecond
	retryClient.RetryWaitMax = 500 * time.Millisecond
	retryClient.Logger = nil

	stdClient := retryClient.StandardClient()
	stdClient.Timeout = 2 * time.Second

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  stdClient,
		logger:  logger,
	}
}

// GetByTitle fetches box office data for a movie title.
func (c *Client) GetByTitle(ctx context.Context, title string) (*Response, error) {
	u, err := url.Parse(c.baseURL + "/boxoffice")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("title", title)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Warn("box office request failed", "title", title, "err", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Info("box office data not found", "title", title)
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Warn("box office unexpected status", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var data Response
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		c.logger.Warn("box office response decode failed", "err", err)
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	return &data, nil
}
