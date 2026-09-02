package muso

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	defaultBaseURL     = "https://api.platform.muso.ai/v1"
	collaboratorsLimit = 50 // Muso rejects collaborator page sizes above 50.
	// Muso allows 300 requests a minute. Pacing at exactly that races the ceiling
	// and the longest rosters get rejected part way through, so leave headroom.
	requestInterval       = time.Minute / 280
	lowRateLimitRemaining = 5
	requestTimeout        = 30 * time.Second
	errorBodyLimit        = 512
)

var (
	ErrNotFound    = errors.New("muso resource not found")
	ErrRateLimited = errors.New("muso rate limit exceeded")
)

type Organisation struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	CollaboratorsCount int    `json:"collaboratorsCount"`
	CreditCount        int    `json:"creditCount"`
}

type CollaboratorPage struct {
	TotalCount int            `json:"totalCount"`
	Items      []Collaborator `json:"items"`
}

// Country is intentionally omitted: Muso rarely supplies it, so country data comes from pb's curated records.
type Collaborator struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	SpotifyID     string   `json:"spotifyId"`
	CommonCredits []string `json:"commonCredits"`
}

const artistCredit = "Artist"

// IsArtist reports whether muso credits this collaborator as a performing artist.
// The collaborators endpoint returns everyone credited on a label's releases,
// songwriters, producers and engineers included; the chart asks only about the
// artists releasing there.
func (c Collaborator) IsArtist() bool {
	for _, credit := range c.CommonCredits {
		if credit == artistCredit {
			return true
		}
	}
	return false
}

type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string

	mu           sync.Mutex
	lastRequest  time.Time
	backoffUntil time.Time
}

func New(apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: requestTimeout}
	}
	return &Client{
		apiKey:     apiKey,
		httpClient: httpClient,
		baseURL:    defaultBaseURL,
	}
}

func (c *Client) SearchOrganisation(ctx context.Context, name string) (Organisation, error) {
	body, err := json.Marshal(struct {
		Keyword string   `json:"keyword"`
		Type    []string `json:"type"`
	}{
		Keyword: name,
		Type:    []string{"organization"},
	})
	if err != nil {
		return Organisation{}, fmt.Errorf("encode muso organisation search: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return Organisation{}, fmt.Errorf("create muso organisation search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var response struct {
		Data struct {
			Organizations struct {
				Items []Organisation `json:"items"`
			} `json:"organizations"`
		} `json:"data"`
	}
	if err := c.do(req, &response); err != nil {
		return Organisation{}, fmt.Errorf("search muso organisation: %w", err)
	}
	if len(response.Data.Organizations.Items) == 0 {
		return Organisation{}, ErrNotFound
	}
	return response.Data.Organizations.Items[0], nil
}

func (c *Client) Collaborators(ctx context.Context, profileID string, offset int) (CollaboratorPage, error) {
	endpoint := c.baseURL + "/profile/" + url.PathEscape(profileID) + "/collaborators"
	query := url.Values{}
	query.Set("limit", strconv.Itoa(collaboratorsLimit))
	query.Set("offset", strconv.Itoa(offset))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return CollaboratorPage{}, fmt.Errorf("create muso collaborators request: %w", err)
	}

	var response struct {
		Data CollaboratorPage `json:"data"`
	}
	if err := c.do(req, &response); err != nil {
		return CollaboratorPage{}, fmt.Errorf("get muso collaborators: %w", err)
	}
	return response.Data, nil
}

func (c *Client) do(req *http.Request, result any) error {
	if err := c.wait(req.Context()); err != nil {
		return err
	}

	req.Header.Set("x-api-key", c.apiKey)
	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	c.observeRateLimit(response.Header)

	switch response.StatusCode {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		// Carry muso's own explanation. A bare status says nothing about which of
		// the request's parameters it objected to.
		detail, _ := io.ReadAll(io.LimitReader(response.Body, errorBodyLimit))
		return fmt.Errorf("muso API returned status %d: %s", response.StatusCode, bytes.TrimSpace(detail))
	}
	if err := json.NewDecoder(response.Body).Decode(result); err != nil {
		return fmt.Errorf("decode muso response: %w", err)
	}
	return nil
}

func (c *Client) wait(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	nextRequest := c.lastRequest.Add(requestInterval)
	if c.backoffUntil.After(nextRequest) {
		nextRequest = c.backoffUntil
	}
	if delay := time.Until(nextRequest); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	c.lastRequest = time.Now()
	return nil
}

func (c *Client) observeRateLimit(header http.Header) {
	remaining, err := strconv.Atoi(header.Get("ratelimit-remaining"))
	if err != nil || remaining > lowRateLimitRemaining {
		return
	}

	reset, err := strconv.ParseInt(header.Get("ratelimit-reset"), 10, 64)
	if err != nil {
		return
	}
	resetAt := time.Now().Add(time.Duration(reset) * time.Second)

	c.mu.Lock()
	if resetAt.After(c.backoffUntil) {
		c.backoffUntil = resetAt
	}
	c.mu.Unlock()
}
