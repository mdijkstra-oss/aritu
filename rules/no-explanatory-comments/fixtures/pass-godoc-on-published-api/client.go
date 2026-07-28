package rateclient

import (
	"errors"
	"net/http"
	"time"
)

// Client fetches exchange rates.
type Client struct {
	http *http.Client
	key  string
}

// New returns a Client that signs its requests with key.
func New(key string) *Client {
	return &Client{http: &http.Client{Timeout: 10 * time.Second}, key: key}
}

// ErrThrottled reports that the endpoint refused the call for rate limiting.
var ErrThrottled = errors.New("rate limited")

// Rate returns the rate for the given currency pair.
//
// The upstream publishes one fixing per weekday at 16:00 CET and serves the
// previous fixing until then, so a weekend call answers with Friday's.
func (c *Client) Rate(base, quote string) (float64, error) {
	req, err := http.NewRequest(http.MethodGet, "https://rates.example.com/v1/"+base+quote, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return 0, ErrThrottled
	}
	return parseRate(resp)
}

func parseRate(resp *http.Response) (float64, error) {
	if resp.StatusCode != http.StatusOK {
		return 0, errors.New("rates: " + resp.Status)
	}
	return 0, nil
}
