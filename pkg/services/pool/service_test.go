package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

func setupTestServer(t *testing.T, handler http.HandlerFunc) (library.Pool, *httptest.Server) {
	server := httptest.NewServer(handler)
	log, _ := logger.New(false)

	baseURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	restClient := &client.Client{
		HttpClient: server.Client(),
		BaseURL:    baseURL,
		AuthToken:  "test-token",
	}

	poolService := New(restClient, log)
	return poolService, server
}

func TestGetPool(t *testing.T) {
	expectedPoolID := uuid.Must(uuid.NewV4())
	expectedPool := payloads.Pool{
		ID:        expectedPoolID,
		NameLabel: "Test Pool",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/pools/"+expectedPoolID.String()))
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(expectedPool)
		assert.NoError(t, err)
	})

	service, server := setupTestServer(t, handler)
	defer server.Close()

	pool, err := service.Get(context.Background(), expectedPoolID)
	assert.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Equal(t, expectedPool.ID, pool.ID)
	assert.Equal(t, expectedPool.NameLabel, pool.NameLabel)
}

func TestGetAllPools(t *testing.T) {
	expectedPools := []payloads.Pool{
		{ID: uuid.Must(uuid.NewV4()), NameLabel: "Pool 1"},
		{ID: uuid.Must(uuid.NewV4()), NameLabel: "Pool 2"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/pools"))
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(expectedPools)
		assert.NoError(t, err)
	})

	service, server := setupTestServer(t, handler)
	defer server.Close()

	pools, err := service.GetAll(context.Background(), 0)
	assert.NoError(t, err)
	assert.NotNil(t, pools)
	assert.Len(t, pools, 2)
	assert.Equal(t, expectedPools[0].NameLabel, pools[0].NameLabel)
	assert.Equal(t, expectedPools[1].NameLabel, pools[1].NameLabel)
}

func TestCreateVM(t *testing.T) {
	poolID := uuid.Must(uuid.NewV4()).String()
	params := payloads.CreateVMParams{
		NameLabel: "New-VM-Test",
		Template:  "Template-uuid",
	}
	expectedVMID := uuid.Must(uuid.NewV4()).String()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, fmt.Sprintf("/pools/%s/vms", poolID)))

		var receivedParams payloads.CreateVMParams
		err := json.NewDecoder(r.Body).Decode(&receivedParams)
		assert.NoError(t, err)
		assert.Equal(t, params.NameLabel, receivedParams.NameLabel)
		assert.Equal(t, params.Template, receivedParams.Template)

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(expectedVMID))
		assert.NoError(t, err)
	})

	service, server := setupTestServer(t, handler)
	defer server.Close()

	vmID, err := service.CreateVM(context.Background(), poolID, params)
	assert.NoError(t, err)
	assert.Equal(t, expectedVMID, vmID)
}

func TestPoolActions(t *testing.T) {
	testCases := []struct {
		name        string
		action      string
		serviceCall func(ctx context.Context, s library.Pool) (string, error)
	}{
		{
			name:   "EmergencyShutdown",
			action: "emergency_shutdown",
			serviceCall: func(ctx context.Context, s library.Pool) (string, error) {
				return s.EmergencyShutdown(ctx)
			},
		},
		{
			name:   "RollingReboot",
			action: "rolling_reboot",
			serviceCall: func(ctx context.Context, s library.Pool) (string, error) {
				return s.RollingReboot(ctx)
			},
		},
		{
			name:   "RollingUpdate",
			action: "rolling_update",
			serviceCall: func(ctx context.Context, s library.Pool) (string, error) {
				return s.RollingUpdate(ctx)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expectedTaskID := "task-" + tc.action

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.True(t, strings.HasSuffix(r.URL.Path, "/pools"))

				var requestBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				assert.NoError(t, err)
				assert.Equal(t, tc.action, requestBody["action"])

				w.Header().Set("Content-Type", "application/json")
				_, err = w.Write([]byte(expectedTaskID))
				assert.NoError(t, err)
			})

			service, server := setupTestServer(t, handler)
			defer server.Close()

			taskID, err := tc.serviceCall(context.Background(), service)
			assert.NoError(t, err)
			assert.Equal(t, expectedTaskID, taskID)
		})
	}
}
