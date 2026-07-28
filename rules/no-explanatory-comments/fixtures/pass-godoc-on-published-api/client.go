package rateclient

import (
	"errors"
	"net/http"
	"time"
)

// Client is safe for concurrent use by any number of callers.
type Client struct {
	http *http.Client
	key  string
}

// New does not validate key; a bad one first surfaces on the next call out.
func New(key string) *Client {
	return &Client{http: &http.Client{Timeout: 10 * time.Second}, key: key}
}

// ErrThrottled comes back when the endpoint refused the call for rate limiting.
var ErrThrottled = errors.New("rate limited")

func (c *Client) QuotePerBase(base, quote string) (float64, error) {
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
