/*
TODO: REMOVE THIS COMMENT. I decided to introduce generic typed functions,
it's going to avoid us having a lot of boilerplate code to make requests
or read response since this already include the marshal and unmarshal
of the payloads. It is also great for avoiding any type and safer type
for the parameters by checking at compile time rather than at runtime.

I am open to suggestions and feedback. We can already see the power of
it in the vm service.
*/
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
)

// Add strong type for the token. Necessary ?
// I don't know but I like it.
type Token string

func (t Token) String() string {
	return string(t)
}

const (
	WebSocketScheme       = "ws"
	SecureWebSocketScheme = "wss"
)

type Client struct {
	/*
		Client has a HttpClient that is constructed with the config.
		I'd prefer to unexport it and hide the implementation details.
		However, for the older version using jsonrpc, it's exported
		and can be used to make requests for missing endpoints.
		When we have a pure REST API, we can remove it since it
		won't be needed anymore.
	*/
	HttpClient *http.Client
	BaseURL    *url.URL
	AuthToken  Token

	RetryMode    core.RetryMode
	RetryMaxTime time.Duration
}

func New(config *config.Config) (*Client, error) {
	if config.Url == "" {
		return nil, errors.New("url is required")
	}

	baseURL, err := url.Parse(config.Url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	if baseURL.Scheme == WebSocketScheme {
		baseURL.Scheme = "http"
	} else if baseURL.Scheme == SecureWebSocketScheme {
		baseURL.Scheme = "https"
	}

	baseURL.Path = path.Join(baseURL.Path, "rest/v0")

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	client := &Client{
		HttpClient:   httpClient,
		BaseURL:      baseURL,
		RetryMode:    config.RetryMode,
		RetryMaxTime: config.RetryMaxTime,
	}

	if config.Token != "" {
		client.AuthToken = Token(config.Token)
	} else if config.Username != "" && config.Password != "" {
		token, err := client.authenticate(config.Username, config.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate: %w", err)
		}
		client.AuthToken = token
	}

	return client, nil
}

func (c *Client) authenticate(username, password string) (Token, error) {
	authURL := *c.BaseURL
	authURL.Path = path.Join(strings.TrimSuffix(c.BaseURL.Path, "rest/v0"), "auth/login")

	payload := map[string]string{
		"username": username,
		"password": password,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, authURL.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to authenticate: %s", string(body))
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "authenticationToken" {
			return Token(cookie.Value), nil
		}
	}

	return "", fmt.Errorf("no auth token found")
}

func (c *Client) do(ctx context.Context, method, endpoint string, params map[string]any, result any) error {
	reqURL := *c.BaseURL
	reqURL.Path = path.Join(reqURL.Path, endpoint)

	var reqBody io.Reader
	if params != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		jsonData, err := json.Marshal(params)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(jsonData)
	} else if params != nil {
		q := reqURL.Query()
		for k, v := range params {
			q.Add(k, fmt.Sprintf("%v", v))
		}
		reqURL.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.AddCookie(&http.Cookie{
		Name:  "authenticationToken",
		Value: c.AuthToken.String(),
	})

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	if result != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return fmt.Errorf("failed to parse response: %w, body: %s", err, string(bodyBytes))
		}
	}

	return nil
}

func (c *Client) get(ctx context.Context, endpoint string, params map[string]any, result any) error {
	return c.do(ctx, "GET", endpoint, params, result)
}

func TypedGet[P any, R any](ctx context.Context, c *Client, endpoint string, params P, result *R) error {
	var paramsMap map[string]any

	if !reflect.ValueOf(params).IsZero() {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &paramsMap); err != nil {
			return err
		}
	}

	return c.get(ctx, endpoint, paramsMap, result)
}

func (c *Client) post(ctx context.Context, endpoint string, params map[string]any, result any) error {
	return c.do(ctx, "POST", endpoint, params, result)
}

func TypedPost[P any, R any](ctx context.Context, c *Client, endpoint string, params P, result *R) error {
	var paramsMap map[string]any
	if !reflect.ValueOf(params).IsZero() {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &paramsMap); err != nil {
			return err
		}
	}
	return c.post(ctx, endpoint, paramsMap, result)
}

func (c *Client) delete(ctx context.Context, endpoint string, params map[string]any, result any) error {
	return c.do(ctx, "DELETE", endpoint, params, result)
}

func TypedDelete[P any, R any](ctx context.Context, c *Client, endpoint string, params P, result *R) error {
	var paramsMap map[string]any
	if !reflect.ValueOf(params).IsZero() {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &paramsMap); err != nil {
			return err
		}
	}
	return c.delete(ctx, endpoint, paramsMap, result)
}
