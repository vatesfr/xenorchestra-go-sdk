package network

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
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

const (
	testNetworkID1        = "550e8400-e29b-41d4-a716-446655440010"
	testNetworkID2        = "550e8400-e29b-41d4-a716-446655440011"
	testNetworkIDNotFound = "550e8400-e29b-41d4-a716-446655440099"
)

var mockNetworks = func() []*payloads.Network {
	return []*payloads.Network{
		{
			ID:              uuid.Must(uuid.FromString(testNetworkID1)),
			NameLabel:       "Pool-wide network associated with eth0",
			NameDescription: "Default network",
			Bridge:          "xenbr0",
			MTU:             1500,
			Automatic:       true,
			Tags:            []string{"production"},
		},
		{
			ID:              uuid.Must(uuid.FromString(testNetworkID2)),
			NameLabel:       "VLAN 100 - Management",
			NameDescription: "Management network",
			Bridge:          "xenbr1",
			MTU:             1500,
			Automatic:       false,
			Tags:            []string{"management"},
		},
	}
}

func findNetworkByID(networkID string) *payloads.Network {
	for _, network := range mockNetworks() {
		if network.ID.String() == networkID {
			return network
		}
	}
	return nil
}

func setupTestServerWithHandler(t *testing.T, handler http.HandlerFunc) (library.Network, *httptest.Server) {
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

func setupTestServer(t *testing.T) (*httptest.Server, library.Network) {
	mux := http.NewServeMux()

	// GET /rest/v0/networks - List all networks
	mux.HandleFunc("GET /rest/v0/networks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		limit := r.URL.Query().Get("limit")
		networks := mockNetworks()

		if limit == "1" && len(networks) > 1 {
			networks = networks[:1]
		}

		if err := json.NewEncoder(w).Encode(networks); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// GET /rest/v0/networks/{id} - Get specific network
	mux.HandleFunc("GET /rest/v0/networks/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		networkID := r.PathValue("id")
		network := findNetworkByID(networkID)

		if network == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(network); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// DELETE /rest/v0/networks/{id} - Delete network
	mux.HandleFunc("DELETE /rest/v0/networks/{id}", func(w http.ResponseWriter, r *http.Request) {
		networkID := r.PathValue("id")

		if network := findNetworkByID(networkID); network == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	// PUT /rest/v0/networks/{id}/tags/{tag} - Add tag to network
	mux.HandleFunc("PUT /rest/v0/networks/{id}/tags/{tag}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		networkID := r.PathValue("id")

		if network := findNetworkByID(networkID); network == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(map[string]bool{"success": true}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// DELETE /rest/v0/networks/{id}/tags/{tag} - Remove tag from network
	mux.HandleFunc("DELETE /rest/v0/networks/{id}/tags/{tag}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		networkID := r.PathValue("id")

		if network := findNetworkByID(networkID); network == nil {
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

func TestNew(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	restClient := &client.Client{
		HttpClient: server.Client(),
		BaseURL:    baseURL,
		AuthToken:  "test-token",
	}

	log, _ := logger.New(true, nil, nil)
	svc := New(restClient, log)

	assert.NotNil(t, svc)
}

func TestGet(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("get existing network", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		network, err := service.Get(context.Background(), networkID)

		assert.NoError(t, err)
		assert.Equal(t, networkID, network.ID)
		assert.Equal(t, "Pool-wide network associated with eth0", network.NameLabel)
		assert.Equal(t, "xenbr0", network.Bridge)
		assert.Equal(t, 1500, network.MTU)
		assert.Contains(t, network.Tags, "production")
	})

	t.Run("get non-existent network", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkIDNotFound))
		_, err := service.Get(context.Background(), networkID)

		assert.Error(t, err)
	})

	t.Run("get network with nil UUID", func(t *testing.T) {
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
			err := json.NewEncoder(w).Encode([]*payloads.Network{})
			assert.NoError(t, err)
		})
		service, server := setupTestServerWithHandler(t, handler)
		defer server.Close()
		networks, err := service.GetAll(context.Background(), limit, filter)
		assert.NoError(t, err)
		assert.NotNil(t, networks)
		assert.True(t, called)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
		service, server := setupTestServerWithHandler(t, handler)
		defer server.Close()
		networks, err := service.GetAll(context.Background(), 0, "")
		assert.Error(t, err)
		assert.Nil(t, networks)
	})

	t.Run("returns error on invalid json", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte("not a json"))
			assert.NoError(t, err)
		})
		service, server := setupTestServerWithHandler(t, handler)
		defer server.Close()
		networks, err := service.GetAll(context.Background(), 0, "")
		assert.Error(t, err)
		assert.Nil(t, networks)
	})

	t.Run("successfully retrieves all networks", func(t *testing.T) {
		server, service := setupTestServer(t)
		defer server.Close()

		networks, err := service.GetAll(context.Background(), 0, "")
		assert.NoError(t, err)
		assert.NotNil(t, networks)
		assert.Len(t, networks, 2)
		assert.Equal(t, "Pool-wide network associated with eth0", networks[0].NameLabel)
		assert.Equal(t, "VLAN 100 - Management", networks[1].NameLabel)
	})

	t.Run("successfully retrieves networks with limit", func(t *testing.T) {
		server, service := setupTestServer(t)
		defer server.Close()

		networks, err := service.GetAll(context.Background(), 1, "")
		assert.NoError(t, err)
		assert.NotNil(t, networks)
		assert.Len(t, networks, 1)
	})
}

func TestDelete(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("successful deletion", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		err := service.Delete(context.Background(), networkID)

		assert.NoError(t, err)
	})

	t.Run("delete non-existent network", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkIDNotFound))
		err := service.Delete(context.Background(), networkID)

		assert.Error(t, err)
	})
}

func TestAddTag(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("successful tag addition", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		err := service.AddTag(context.Background(), networkID, "new-tag")

		assert.NoError(t, err)
	})

	t.Run("add tag with empty string", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		err := service.AddTag(context.Background(), networkID, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tag cannot be empty")
	})

	t.Run("add tag with non-existent network", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkIDNotFound))
		err := service.AddTag(context.Background(), networkID, "tag1")

		assert.Error(t, err)
	})
}

func TestRemoveTag(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("successful tag removal", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		err := service.RemoveTag(context.Background(), networkID, "production")

		assert.NoError(t, err)
	})

	t.Run("remove tag with empty string", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		err := service.RemoveTag(context.Background(), networkID, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tag cannot be empty")
	})

	t.Run("remove tag with non-existent network", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkIDNotFound))
		err := service.RemoveTag(context.Background(), networkID, "tag1")

		assert.Error(t, err)
	})
}
