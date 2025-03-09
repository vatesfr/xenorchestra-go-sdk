package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
)

var ctx = context.Background()

func TestNew(t *testing.T) {
	_, err := New(&config.Config{Url: "://invalid-url"})
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}

	_, err = New(&config.Config{Url: "http://example.com"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestAuthenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			var creds struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			json.NewDecoder(r.Body).Decode(&creds)

			if creds.Username == "testuser" && creds.Password == "testpass" {
				http.SetCookie(w, &http.Cookie{
					Name:  "authenticationToken",
					Value: "test-token",
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

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if client.AuthToken != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", client.AuthToken)
	}

	_, err = New(&config.Config{
		Url:      server.URL,
		Username: "wrong",
		Password: "wrong",
	})

	if err == nil {
		t.Error("Expected authentication error, got nil")
	}
}

func TestDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("authenticationToken")
		if err != nil || cookie.Value != "test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/rest/v0/test" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result":"success"}`))
			return
		}

		if r.URL.Path == "/rest/v0/test" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"123"}`))
			return
		}

		if r.URL.Path == "/rest/v0/error" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"test error"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
	}

	var getResult struct {
		Result string `json:"result"`
	}
	err := client.get(ctx, "test", nil, &getResult)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if getResult.Result != "success" {
		t.Errorf("Expected result 'success', got '%s'", getResult.Result)
	}

	var postResult struct {
		ID string `json:"id"`
	}
	err = client.post(ctx, "test", map[string]interface{}{"key": "value"}, &postResult)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if postResult.ID != "123" {
		t.Errorf("Expected ID '123', got '%s'", postResult.ID)
	}

	err = client.post(ctx, "error", nil, nil)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestTypedGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v0/test" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name":"test-item","value":123}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
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

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Name != "test-item" || result.Value != 123 {
		t.Errorf("Expected result {Name:'test-item', Value:123}, got %+v", result)
	}
}
