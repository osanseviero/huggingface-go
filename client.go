package huggingface

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://huggingface.co"

// HubClient represents a client for interacting with the Hugging Face Hub.
type HubClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Auth       *Auth
}

// NewHubClient creates a new Hugging Face client.
func NewHubClient() (*HubClient, error) {
	auth, err := NewAuthFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create auth: %w", err)
	}

	return &HubClient{
		BaseURL: defaultBaseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Auth: auth,
	}, nil
}


// doRequest handles HTTP requests
func (c *HubClient) doRequest(method, endpoint string, body interface{}, headers map[string]string) (*http.Response, error) {
	reqBody, err := c.prepareRequestBody(body)
	if err != nil {
		return nil, err
	}

	fullURL := c.prepareFullURL(endpoint)

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s %s: %w", method, fullURL, err)
	}

	c.setHeaders(req, headers)

	return c.HTTPClient.Do(req)
}

func (c *HubClient) prepareRequestBody(body interface{}) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return bytes.NewBuffer(jsonData), nil
}

func (c *HubClient) prepareFullURL(endpoint string) string {
	if strings.HasPrefix(endpoint, c.BaseURL) {
		return endpoint
	}
	return c.BaseURL + endpoint
}

func (c *HubClient) setHeaders(req *http.Request, headers map[string]string) {
	req.Header.Set("Authorization", c.Auth.Header())
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

// parseResponse parses the HTTP response
func parseResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if v == nil {
			return nil
		}
		return json.NewDecoder(resp.Body).Decode(v)
	}

	// Handle errors
	var errResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return NewAPIError(resp.StatusCode, resp.Status)
	}

	message, ok := errResp["message"].(string)
	if !ok {
		message = fmt.Sprintf("status: %s", resp.Status)
	}

	return NewAPIError(resp.StatusCode, message)
}