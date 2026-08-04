package decodo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type authType int

const (
	authTypeBasic authType = iota
)

type httpClientConfig struct {
	baseURL           string
	authType          authType
	authToken         string
	timeoutMs         int
	integrationHeader string
}

type httpClient struct {
	config httpClientConfig
	client *http.Client
}

func newHTTPClient(config httpClientConfig) *httpClient {
	transport := &http.Transport{
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}
	return &httpClient{
		config: config,
		client: &http.Client{Transport: transport},
	}
}

type errorResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Errors  []interface{} `json:"errors"`
}

func (c *httpClient) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	reqURL := strings.TrimRight(c.config.baseURL, "/") + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(c.config.timeoutMs)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if c.config.authType == authTypeBasic {
		req.Header.Set("Authorization", "Basic "+c.config.authToken)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-integration", c.config.integrationHeader)

	resp, err := c.client.Do(req)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
			return nil, &TimeoutError{Msg: fmt.Sprintf("request to %s timed out after %dms", path, c.config.timeoutMs)}
		}
		if ctx.Err() == context.Canceled {
			return nil, &CancellationError{Msg: fmt.Sprintf("request to %s was cancelled", path)}
		}
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == 204 {
		return nil, nil
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}

	var errResp errorResponse
	_ = json.Unmarshal(respBody, &errResp)

	message := errResp.Message
	if message == "" {
		message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return nil, mapHTTPError(resp.StatusCode, message, errResp.Status, errResp.Errors)
}

func (c *httpClient) post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.request(ctx, http.MethodPost, path, body)
}

func (c *httpClient) get(ctx context.Context, path string, query map[string]string) ([]byte, error) {
	if len(query) > 0 {
		params := url.Values{}
		for k, v := range query {
			params.Set(k, v)
		}
		if strings.Contains(path, "?") {
			path = path + "&" + params.Encode()
		} else {
			path = path + "?" + params.Encode()
		}
	}
	return c.request(ctx, http.MethodGet, path, nil)
}
