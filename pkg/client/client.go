package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/takayoshi/vast-cli/pkg/config"
)

var apiVersionRegex = regexp.MustCompile(`^/api/v\d+/`)

// Client manages communication with the Vast.ai API
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	RetryCount int
	Explain    bool
	Curl       bool
	Timeout    time.Duration
}

// Option configures a Client
type Option func(*Client)

func WithAPIKey(key string) Option {
	return func(c *Client) {
		c.APIKey = key
	}
}

func WithBaseURL(urlStr string) Option {
	return func(c *Client) {
		c.BaseURL = strings.TrimRight(urlStr, "/")
	}
}

func WithExplain(explain bool) Option {
	return func(c *Client) {
		c.Explain = explain
	}
}

func WithCurl(curl bool) Option {
	return func(c *Client) {
		c.Curl = curl
	}
}

func WithRetry(retry int) Option {
	return func(c *Client) {
		c.RetryCount = retry
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.Timeout = d
	}
}

// NewClient initializes a new Vast.ai API client
func NewClient(opts ...Option) *Client {
	c := &Client{
		BaseURL:    config.DefaultServerURL,
		RetryCount: 3,
		Timeout:    120 * time.Second,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.Timeout > 0 {
		c.HTTPClient.Timeout = c.Timeout
	}
	return c
}

func (c *Client) buildURL(subpath string, query url.Values) string {
	if !apiVersionRegex.MatchString(subpath) {
		if !strings.HasPrefix(subpath, "/") {
			subpath = "/" + subpath
		}
		subpath = "/api/v0" + subpath
	}
	fullURL := c.BaseURL + subpath
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	return fullURL
}

func (c *Client) printExplain(method, reqURL string, headers http.Header, body []byte) {
	fmt.Printf("\nℹ️  Prepared Request:\n")
	fmt.Printf("%s %s\n", method, reqURL)
	hJSON, _ := json.MarshalIndent(headers, "", " ")
	fmt.Printf("Headers: %s\n", string(hJSON))
	if len(body) > 0 {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "", " "); err == nil {
			fmt.Printf("Body: %s\n", pretty.String())
		} else {
			fmt.Printf("Body: %s\n", string(body))
		}
	} else {
		fmt.Printf("Body: null\n")
	}
	fmt.Printf("____________________________________________________________________________________________________\n\n")
}

func (c *Client) printCurl(method, reqURL string, headers http.Header, body []byte) {
	parts := []string{"curl"}
	if method != http.MethodGet || len(body) > 0 {
		parts = append(parts, fmt.Sprintf("-X %s", method))
	}
	for k, v := range headers {
		parts = append(parts, fmt.Sprintf("-H '%s: %s'", k, strings.Join(v, ", ")))
	}
	if len(body) > 0 {
		parts = append(parts, fmt.Sprintf("-d '%s'", string(body)))
	}
	parts = append(parts, fmt.Sprintf("'%s'", reqURL))
	fmt.Println(strings.Join(parts, " \\\n  "))
}

// Request executes an HTTP request with automatic retry and JSON body
func (c *Client) Request(ctx context.Context, method, subpath string, query url.Values, reqBody interface{}) ([]byte, int, error) {
	reqURL := c.buildURL(subpath, query)

	var bodyBytes []byte
	if reqBody != nil {
		var err error
		bodyBytes, err = json.Marshal(reqBody)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	headers := http.Header{}
	headers.Set("User-Agent", fmt.Sprintf("vastai-go/%s", config.Version))
	if c.APIKey != "" {
		headers.Set("Authorization", "Bearer "+c.APIKey)
	}
	if len(bodyBytes) > 0 || method == http.MethodPost || method == http.MethodPut {
		headers.Set("Content-Type", "application/json")
	}

	if c.Explain {
		c.printExplain(method, reqURL, headers, bodyBytes)
	}

	if c.Curl {
		c.printCurl(method, reqURL, headers, bodyBytes)
		return nil, 0, nil
	}

	retryLimit := c.RetryCount
	if retryLimit < 1 {
		retryLimit = 1
	}

	delay := 150 * time.Millisecond
	var lastErr error
	var lastStatusCode int

	for i := 0; i < retryLimit; i++ {
		var bodyReader io.Reader
		if len(bodyBytes) > 0 {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
		}
		req.Header = headers.Clone()

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			if i < retryLimit-1 {
				time.Sleep(delay)
				delay = time.Duration(float64(delay) * 1.5)
				continue
			}
			return nil, 0, fmt.Errorf("request failed after %d attempts: %w", retryLimit, lastErr)
		}

		lastStatusCode = resp.StatusCode
		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if readErr != nil {
			lastErr = readErr
			if i < retryLimit-1 {
				time.Sleep(delay)
				delay = time.Duration(float64(delay) * 1.5)
				continue
			}
			return nil, lastStatusCode, fmt.Errorf("failed to read response body: %w", readErr)
		}

		// Retryable HTTP status codes
		if (resp.StatusCode == 429 || resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504) && i < retryLimit-1 {
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * 1.5)
			continue
		}

		if resp.StatusCode >= 400 {
			var errResp struct {
				Msg   string `json:"msg"`
				Error string `json:"error"`
			}
			msg := strings.TrimSpace(string(respBody))
			if json.Unmarshal(respBody, &errResp) == nil {
				if errResp.Msg != "" {
					msg = errResp.Msg
				} else if errResp.Error != "" {
					msg = errResp.Error
				}
			}
			return respBody, resp.StatusCode, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, msg)
		}

		return respBody, resp.StatusCode, nil
	}

	return nil, lastStatusCode, fmt.Errorf("request failed: %w", lastErr)
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, subpath string, query url.Values) ([]byte, error) {
	data, _, err := c.Request(ctx, http.MethodGet, subpath, query, nil)
	return data, err
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, subpath string, query url.Values, body interface{}) ([]byte, error) {
	data, _, err := c.Request(ctx, http.MethodPost, subpath, query, body)
	return data, err
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, subpath string, query url.Values, body interface{}) ([]byte, error) {
	data, _, err := c.Request(ctx, http.MethodPut, subpath, query, body)
	return data, err
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, subpath string, query url.Values, body interface{}) ([]byte, error) {
	data, _, err := c.Request(ctx, http.MethodDelete, subpath, query, body)
	return data, err
}

// PollResultURL polls an asynchronous result URL until complete (used for logs and execute)
func (c *Client) PollResultURL(ctx context.Context, resultURL string, retries int, delay time.Duration) (string, error) {
	if retries <= 0 {
		retries = 30
	}
	if delay <= 0 {
		delay = 300 * time.Millisecond
	}

	pollClient := &http.Client{Timeout: 10 * time.Second}
	for i := 0; i < retries; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		time.Sleep(delay)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, resultURL, nil)
		if err != nil {
			return "", err
		}

		resp, err := pollClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			body, rErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if rErr == nil {
				return string(body), nil
			}
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	return "", fmt.Errorf("result not ready after polling %s", resultURL)
}
