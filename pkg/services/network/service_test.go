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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	mock "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library/mock"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

const (
	testNetworkID1        = "550e8400-e29b-41d4-a716-446655440010"
	testNetworkID2        = "550e8400-e29b-41d4-a716-446655440011"
	testNetworkIDNotFound = "550e8400-e29b-41d4-a716-446655440099"
	testTokenValue        = "test-token"
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

func findNetworkByID(networkID uuid.UUID) *payloads.Network {
	for _, network := range mockNetworks() {
		if network.ID == networkID {
			return network
		}
	}
	return nil
}

func setupTestServerWithHandler(
	t *testing.T, handler http.HandlerFunc) (library.Network, *httptest.Server, *mock.MockTask) {
	t.Helper()
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
		AuthToken:  testTokenValue,
	}
	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)

	mockService := New(restClient, mockTask, nil, log)
	return mockService, server, mockTask
}

func setupTestServer(t *testing.T) (*httptest.Server, library.Network, *mock.MockTask) {
	t.Helper()
	mux := http.NewServeMux()

	// GET /rest/v0/networks - List all networks
	mux.HandleFunc("GET /rest/v0/networks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockNetworks()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// GET /rest/v0/networks/{id} - Get specific network
	mux.HandleFunc("GET /rest/v0/networks/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		networkID, err := uuid.FromString(r.PathValue("id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
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
		networkID, err := uuid.FromString(r.PathValue("id"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if network := findNetworkByID(networkID); network == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	// GET /rest/v0/networks/{id}/tasks - Get tasks for a network
	mux.HandleFunc("GET /rest/v0/networks/{id}/tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := uuid.FromString(r.PathValue("id")); err != nil {
			http.NotFound(w, r)
			return
		}
		tasks := []*payloads.Task{
			{ID: "task-001", Status: payloads.Success},
			{ID: "task-002", Status: payloads.Failure},
		}
		if err := json.NewEncoder(w).Encode(tasks); err != nil {
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
	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)
	return server, New(restClient, mockTask, nil, log), mockTask
}

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Url:   "http://localhost",
		Token: testTokenValue,
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)

	log, _ := logger.New(true, nil, nil)
	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)
	svc := New(c, mockTask, nil, log)

	assert.NotNil(t, svc)
}

func TestGet(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	t.Run("get existing network", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		network, err := service.Get(t.Context(), networkID)

		assert.NoError(t, err)
		require.NotNil(t, network)
		net := mockNetworks()[0]
		assert.Equal(t, networkID, network.ID)
		assert.Equal(t, net.NameLabel, network.NameLabel)
		assert.Equal(t, net.Bridge, network.Bridge)
		assert.Equal(t, net.MTU, network.MTU)
		assert.Contains(t, network.Tags, "production")
	})

	t.Run("get non-existent network", func(t *testing.T) {
		networkID := uuid.Must(uuid.FromString(testNetworkIDNotFound))
		result, err := service.Get(t.Context(), networkID)

		assert.Error(t, err)
		assert.Nil(t, result)
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
			if err := json.NewEncoder(w).Encode([]*payloads.Network{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		service, server, _ := setupTestServerWithHandler(t, handler)
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
		service, server, _ := setupTestServerWithHandler(t, handler)
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
		service, server, _ := setupTestServerWithHandler(t, handler)
		defer server.Close()
		networks, err := service.GetAll(context.Background(), 0, "")
		assert.Error(t, err)
		assert.Nil(t, networks)
	})

	t.Run("successfully retrieves all networks", func(t *testing.T) {
		server, service, _ := setupTestServer(t)
		defer server.Close()

		networks, err := service.GetAll(context.Background(), 0, "")
		require.NoError(t, err)
		require.NotNil(t, networks)
		require.Len(t, networks, 2)
		assert.Equal(t, uuid.Must(uuid.FromString(testNetworkID1)), networks[0].ID)
		assert.Equal(t, uuid.Must(uuid.FromString(testNetworkID2)), networks[1].ID)
	})
}

func TestDelete(t *testing.T) {
	server, service, _ := setupTestServer(t)
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

func TestGetTasks(t *testing.T) {
	t.Run("passes limit and filter parameters", func(t *testing.T) {
		limit := 5
		filter := "status:failure"
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, http.MethodGet, r.Method)
			values := r.URL.Query()
			assert.Equal(t, fmt.Sprintf("%d", limit), values.Get("limit"))
			assert.Equal(t, filter, values.Get("filter"))
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]*payloads.Task{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		service, server, _ := setupTestServerWithHandler(t, handler)
		defer server.Close()

		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		tasks, err := service.GetTasks(context.Background(), networkID, limit, filter)

		assert.NoError(t, err)
		assert.NotNil(t, tasks)
		assert.True(t, called)
	})

	t.Run("does not send limit param when zero", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			values := r.URL.Query()
			assert.Empty(t, values.Get("limit"))
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]*payloads.Task{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		service, server, _ := setupTestServerWithHandler(t, handler)
		defer server.Close()

		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		_, err := service.GetTasks(context.Background(), networkID, 0, "")
		assert.NoError(t, err)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
		service, server, _ := setupTestServerWithHandler(t, handler)
		defer server.Close()

		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		tasks, err := service.GetTasks(context.Background(), networkID, 0, "")

		assert.Error(t, err)
		assert.Nil(t, tasks)
	})

	t.Run("successfully retrieves tasks for a network", func(t *testing.T) {
		server, service, _ := setupTestServer(t)
		defer server.Close()

		networkID := uuid.Must(uuid.FromString(testNetworkID1))
		tasks, err := service.GetTasks(context.Background(), networkID, 0, "")

		assert.NoError(t, err)
		require.NotNil(t, tasks)
		assert.Len(t, tasks, 2)
		assert.Equal(t, "task-001", tasks[0].ID)
		assert.Equal(t, payloads.Success, tasks[0].Status)
		assert.Equal(t, "task-002", tasks[1].ID)
		assert.Equal(t, payloads.Failure, tasks[1].Status)
	})
}

func TestCreate(t *testing.T) {
	cfg := &config.Config{
		Url:   "http://localhost",
		Token: testTokenValue,
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)

	log, _ := logger.New(true, nil, nil)
	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)
	mockPool := mock.NewMockPool(ctrl)
	mockPool.EXPECT().CreateNetwork(
		gomock.Any(), gomock.Any(), gomock.Any()).Return(uuid.Must(uuid.FromString(testNetworkID1)), nil).Times(1)
	svc := New(c, mockTask, mockPool, log)

	require.NotNil(t, svc)

	_, err = svc.Create(t.Context(), uuid.Must(uuid.NewV4()), payloads.CreateNetworkParams{
		Name: "Test network",
		Vlan: 0,
		Pif:  uuid.Must(uuid.NewV4()),
	})
	assert.NoError(t, err)
}
