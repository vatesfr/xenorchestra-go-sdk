package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
)

var ctx = context.Background()

const (
	restPath       = "/rest/v0"
	testTokenValue = "test-token"
	testToken      = "abc"
)

func TestNew(t *testing.T) {
	t.Run("InvalidURL", func(t *testing.T) {
		_, err := New(&config.Config{Url: "://invalid-url", Token: testToken})
		assert.Error(t, err)
	})

	t.Run("ValidTokenAuth", func(t *testing.T) {
		_, err := New(&config.Config{Url: "http://example.com", Token: testToken})
		assert.NoError(t, err)
	})

	t.Run("DefaultClientTimeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := New(&config.Config{
			Url:   server.URL,
			Token: testToken,
			// ClientTimeout not set – should default to 30s
		})
		require.NoError(t, err)
		assert.Equal(t, 30*time.Second, client.HttpClient.Timeout)
	})

	t.Run("CustomClientTimeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		for _, tc := range []struct {
			name    string
			timeout time.Duration
		}{
			{"1min", 1 * time.Minute},
			{"5s", 5 * time.Second},
		} {
			t.Run(tc.name, func(t *testing.T) {
				client, err := New(&config.Config{
					Url:           server.URL,
					Token:         testToken,
					ClientTimeout: tc.timeout,
				})
				require.NoError(t, err)
				assert.Equal(t, tc.timeout, client.HttpClient.Timeout)
			})
		}
	})
}

func TestAuthenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			var creds struct {
				Username string `json:"username"`
				Password string `json:"password"` //gosec:disable G117
			}
			_ = json.NewDecoder(r.Body).Decode(&creds)

			if creds.Username == "testuser" && creds.Password == "testpass" {
				http.SetCookie(w, &http.Cookie{
					Name:     authCookieName,
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
					Secure:   true,
					Value:    testTokenValue,
				})
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := New(&config.Config{
		Url:      server.URL,
		Username: "testuser",
		Password: "testpass",
	})
	require.NoError(t, err)
	assert.EqualValues(t, testTokenValue, client.AuthToken)

	_, err = New(&config.Config{
		Url:      server.URL,
		Username: "wrong",
		Password: "wrong",
	})
	assert.Error(t, err)
}

func TestDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(authCookieName)
		if err != nil || cookie.Value != testTokenValue {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == restPath+"/test" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"success"}`))
			return
		}

		if r.URL.Path == restPath+"/test" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"123"}`))
			return
		}

		if r.URL.Path == restPath+"/error" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"test error"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: httpScheme, Host: server.URL[7:], Path: restPath},
		AuthToken:  testTokenValue,
	}

	var getResult struct {
		Result string `json:"result"`
	}
	err := client.get(ctx, "test", nil, &getResult)
	require.NoError(t, err)
	assert.Equal(t, "success", getResult.Result)

	var postResult struct {
		ID string `json:"id"`
	}
	err = client.post(ctx, "test", map[string]interface{}{"key": "value"}, &postResult)
	require.NoError(t, err)
	assert.Equal(t, "123", postResult.ID)

	err = client.post(ctx, "error", nil, nil)
	assert.Error(t, err)
}

func TestTypedGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == restPath+"/test" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name":"test-item","value":123}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: httpScheme, Host: server.URL[7:], Path: restPath},
		AuthToken:  testTokenValue,
	}

	// Define the request and response types
	type TestParams struct {
		Filter string `json:"filter"`
	}

	type TestResult struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	params := TestParams{Filter: "active"}
	var result TestResult

	err := TypedGet(ctx, client, "test", &params, &result)
	require.NoError(t, err)
	assert.Equal(t, "test-item", result.Name)
	assert.Equal(t, 123, result.Value)
}
