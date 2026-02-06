package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

// This is a basic test to ensure the service can be instantiated.
func TestNew(t *testing.T) {
	cfg := &config.Config{
		Url:   "http://localhost",
		Token: "test-token",
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)

	log, _ := logger.New(true, nil, nil)
	svc := New(c, log)

	assert.NotNil(t, svc)
}

func TestHostService_Get_InvalidUUID(t *testing.T) {
	// This test mainly checks that the code compiles and imports are correct
	cfg := &config.Config{
		Url:   "http://localhost",
		Token: "test-token",
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)

	log, _ := logger.New(true, nil, nil)
	svc := New(c, log)

	_, err = svc.Get(context.Background(), uuid.Nil)
	// Since we don't have a real server, we expect an error or it to try to connect
	// In this unit test setup without mocks, we mainly ensure type safety
	assert.Error(t, err)
}

const (
	testHostID1        = "550e8400-e29b-41d4-a716-446655440000"
	testHostID2        = "550e8400-e29b-41d4-a716-446655440001"
	testHostIDNotFound = "550e8400-e29b-41d4-a716-446655440099"
)

// Use a function to generate mock hosts to avoid issues with mutable slices in tests
var mockHosts = func() []*payloads.Host {
	return []*payloads.Host{
		{
			ID:           uuid.Must(uuid.FromString(testHostID1)),
			NameLabel:    "host1.example.com",
			ProductBrand: "XCP-ng",
			Tags:         []string{"tag1"},
		},
		{
			ID:           uuid.Must(uuid.FromString(testHostID2)),
			NameLabel:    "host2.example.com",
			ProductBrand: "XCP-ng",
			Tags:         []string{"tag2"},
		},
	}
}

func findHostByID(hostID string) *payloads.Host {
	for _, host := range mockHosts() {
		if host.ID.String() == hostID {
			return host
		}
	}
	return nil
}

func setupTestServerWithHandler(t *testing.T, handler http.HandlerFunc) (library.Host, *httptest.Server) {
	server := httptest.NewServer(handler)
	log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	restClient := &client.Client{
		HttpClient: server.Client(),
		BaseURL:    baseURL,
		AuthToken:  "test-token",
	}

	mockService := New(restClient, log)
	return mockService, server
}

func setupTestServer(t *testing.T) (*httptest.Server, library.Host) {
	mux := http.NewServeMux()

	// GET /rest/v0/hosts - List all hosts
	mux.HandleFunc("GET /rest/v0/hosts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		limit := r.URL.Query().Get("limit")
		hosts := mockHosts()

		if limit == "1" && len(hosts) > 1 {
			hosts = hosts[:1]
		}

		if err := json.NewEncoder(w).Encode(hosts); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// GET /rest/v0/hosts/{id} - Get specific host
	mux.HandleFunc("GET /rest/v0/hosts/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		hostID := r.PathValue("id")
		host := findHostByID(hostID)

		if host == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(host); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// PUT /rest/v0/hosts/{id}/tags/{tag} - Add tag to host
	mux.HandleFunc("PUT /rest/v0/hosts/{id}/tags/{tag}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		hostID := r.PathValue("id")

		if host := findHostByID(hostID); host == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// DELETE /rest/v0/hosts/{id}/tags/{tag} - Remove tag from host
	mux.HandleFunc("DELETE /rest/v0/hosts/{id}/tags/{tag}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		hostID := r.PathValue("id")

		if host := findHostByID(hostID); host == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	server := httptest.NewServer(mux)

	restClient := &client.Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
	}

	log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	return server, New(restClient, log)
}

func TestGet(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("get existing host", func(t *testing.T) {
		hostID := uuid.Must(uuid.FromString(testHostID1))
		host, err := service.Get(context.Background(), hostID)

		assert.NoError(t, err)
		assert.Equal(t, hostID, host.ID)
		assert.Equal(t, "host1.example.com", host.NameLabel)
		assert.Equal(t, "XCP-ng", host.ProductBrand)
		assert.Contains(t, host.Tags, "tag1")
	})

	t.Run("get non-existent host", func(t *testing.T) {
		hostID := uuid.Must(uuid.FromString(testHostIDNotFound))
		_, err := service.Get(context.Background(), hostID)

		assert.Error(t, err)
	})

	t.Run("get host with nil UUID", func(t *testing.T) {
		_, err := service.Get(context.Background(), uuid.Nil)

		assert.Error(t, err)
	})
}

func TestGetAll(t *testing.T) {
	t.Run("passes limit and filter parameters", func(t *testing.T) {
		limit := 42
		filter := "filter-to-check"
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, http.MethodGet, r.Method)
			values := r.URL.Query()
			assert.Equal(t, fmt.Sprintf("%d", limit), values.Get("limit"))
			assert.Equal(t, filter, values.Get("filter"))
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode([]*payloads.Host{})
			assert.NoError(t, err)
		})
		service, server := setupTestServerWithHandler(t, handler)
		defer server.Close()
		hosts, err := service.GetAll(context.Background(), limit, filter)
		assert.NoError(t, err)
		assert.NotNil(t, hosts)
		assert.True(t, called)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
		service, server := setupTestServerWithHandler(t, handler)
		defer server.Close()
		hosts, err := service.GetAll(context.Background(), 0, "")
		assert.Error(t, err)
		assert.Nil(t, hosts)
	})

	t.Run("returns error on invalid json", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte("not a json"))
			assert.NoError(t, err)
		})
		service, server := setupTestServerWithHandler(t, handler)
		defer server.Close()
		hosts, err := service.GetAll(context.Background(), 0, "")
		assert.Error(t, err)
		assert.Nil(t, hosts)
	})

	t.Run("successfully retrieves all hosts", func(t *testing.T) {
		server, service := setupTestServer(t)
		defer server.Close()

		hosts, err := service.GetAll(context.Background(), 0, "")
		assert.NoError(t, err)
		assert.NotNil(t, hosts)
		assert.Len(t, hosts, 2)
		assert.Equal(t, "host1.example.com", hosts[0].NameLabel)
		assert.Equal(t, "host2.example.com", hosts[1].NameLabel)
	})

	t.Run("successfully retrieves hosts with limit", func(t *testing.T) {
		server, service := setupTestServer(t)
		defer server.Close()

		hosts, err := service.GetAll(context.Background(), 1, "")
		assert.NoError(t, err)
		assert.NotNil(t, hosts)
		assert.Len(t, hosts, 1)
	})
}

func TestAddTag(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("successful tag addition", func(t *testing.T) {
		hostID := uuid.Must(uuid.FromString(testHostID1))
		err := service.AddTag(context.Background(), hostID, "new-tag")

		assert.NoError(t, err)
	})

	t.Run("add tag with empty string", func(t *testing.T) {
		hostID := uuid.Must(uuid.FromString(testHostID1))
		err := service.AddTag(context.Background(), hostID, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tag cannot be empty")
	})

	t.Run("add tag with unexistent host", func(t *testing.T) {
		hostID := uuid.Must(uuid.FromString(testHostIDNotFound))
		err := service.AddTag(context.Background(), hostID, "tag1")

		assert.Error(t, err)
	})
}

func TestRemoveTag(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("successful tag removal", func(t *testing.T) {
		hostID := uuid.Must(uuid.FromString(testHostID1))
		err := service.RemoveTag(context.Background(), hostID, "tag1")

		assert.NoError(t, err)
	})

	t.Run("remove tag with empty string", func(t *testing.T) {
		hostID := uuid.Must(uuid.FromString(testHostID1))
		err := service.RemoveTag(context.Background(), hostID, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tag cannot be empty")
	})

	t.Run("remove tag with unexistent host", func(t *testing.T) {
		hostID := uuid.Must(uuid.FromString(testHostIDNotFound))
		err := service.RemoveTag(context.Background(), hostID, "tag1")

		assert.Error(t, err)
	})
}
