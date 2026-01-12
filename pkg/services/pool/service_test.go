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
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	mock "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library/mock"
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

	// Create mock controller and task mock
	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)

	poolService := New(restClient, mockTask, log)
	return poolService, server
}

func TestGetPool(t *testing.T) {
	t.Run("returns error on http error", func(t *testing.T) {
		expectedPoolID := uuid.Must(uuid.NewV4())
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
		service, server := setupTestServer(t, handler)
		defer server.Close()
		pool, err := service.Get(context.Background(), expectedPoolID)
		assert.Error(t, err)
		assert.Nil(t, pool)
	})

	t.Run("returns error on invalid json", func(t *testing.T) {
		expectedPoolID := uuid.Must(uuid.NewV4())
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte("not a json"))
			assert.NoError(t, err)
		})
		service, server := setupTestServer(t, handler)
		defer server.Close()
		pool, err := service.Get(context.Background(), expectedPoolID)
		assert.Error(t, err)
		assert.Nil(t, pool)
	})

	t.Run("successfully retrieves pool", func(t *testing.T) {
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
	})
}

func TestGetAllPools(t *testing.T) {
	t.Run("passes limit parameter", func(t *testing.T) {
		limit := 42
		filter := "filter-to-check"
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, http.MethodGet, r.Method)
			// Vérifie que le paramètre limit est bien dans l'URL
			values := r.URL.Query()
			assert.Equal(t, fmt.Sprintf("%d", limit), values.Get("limit"))
			assert.Equal(t, filter, values.Get("filter"))
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode([]payloads.Pool{})
			assert.NoError(t, err)
		})
		service, server := setupTestServer(t, handler)
		defer server.Close()
		pools, err := service.GetAll(context.Background(), limit, filter)
		assert.NoError(t, err)
		assert.NotNil(t, pools)
		assert.True(t, called)
	})
	t.Run("returns error on http error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
		service, server := setupTestServer(t, handler)
		defer server.Close()
		pools, err := service.GetAll(context.Background(), 0, "")
		assert.Error(t, err)
		assert.Nil(t, pools)
	})

	t.Run("returns error on invalid json", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte("not a json"))
			assert.NoError(t, err)
		})
		service, server := setupTestServer(t, handler)
		defer server.Close()
		pools, err := service.GetAll(context.Background(), 0, "")
		assert.Error(t, err)
		assert.Nil(t, pools)
	})

	t.Run("successfully retrieves all pools", func(t *testing.T) {
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

		pools, err := service.GetAll(context.Background(), 0, "")
		assert.NoError(t, err)
		assert.NotNil(t, pools)
		assert.Len(t, pools, 2)
		assert.Equal(t, expectedPools[0].NameLabel, pools[0].NameLabel)
		assert.Equal(t, expectedPools[1].NameLabel, pools[1].NameLabel)
	})
}

func TestCreateResource(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		poolID := uuid.Must(uuid.NewV4())
		expectedID := uuid.Must(uuid.NewV4())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockTask := mock.NewMockTask(ctrl)
		mockTask.EXPECT().HandleTaskResponse(gomock.Any(), "task-response", true).Return(&payloads.Task{
			Status: payloads.Success,
			Result: payloads.Result{ID: expectedID},
		}, true, nil)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			// read and assert request body contains the params
			var vm payloads.CreateVMParams
			err := json.NewDecoder(r.Body).Decode(&vm)
			assert.NoError(t, err)
			assert.Equal(t, "test-vm", vm.NameLabel)
			_, _ = w.Write([]byte("{\"taskId\":\"task-response\"}"))
		})

		poolService, server := setupTestServer(t, handler)
		defer server.Close()
		s := poolService.(*Service)
		s.taskService = mockTask

		params := payloads.CreateVMParams{
			NameLabel: "test-vm",
		}
		gotID, err := s.CreateVM(context.Background(), poolID, params)
		assert.NoError(t, err)
		assert.Equal(t, expectedID, gotID)
	})

	t.Run("http error", func(t *testing.T) {
		poolID := uuid.Must(uuid.NewV4())

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "err", http.StatusInternalServerError)
		})
		poolService, server := setupTestServer(t, handler)
		defer server.Close()
		s := poolService.(*Service)

		gotID, err := s.createResource(context.Background(), poolID, "vm", nil)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, gotID)
	})

	t.Run("task handler error", func(t *testing.T) {
		poolID := uuid.Must(uuid.NewV4())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockTask := mock.NewMockTask(ctrl)
		mockTask.EXPECT().HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: "task-response"}, true).
			Return(nil, fmt.Errorf("boom"))

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("{\"taskId\":\"task-response\"}"))
		})
		poolService, server := setupTestServer(t, handler)
		defer server.Close()
		s := poolService.(*Service)
		s.taskService = mockTask

		gotID, err := s.createResource(context.Background(), poolID, "vm", nil)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, gotID)
	})

	t.Run("unexpected response (not a task)", func(t *testing.T) {
		poolID := uuid.Must(uuid.NewV4())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockTask := mock.NewMockTask(ctrl)
		// Return isTask=false
		mockTask.EXPECT().HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: "not-a-task"}, true).
			Return(nil, nil)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("{\"taskId\":\"not-a-task\"}"))
		})
		poolService, server := setupTestServer(t, handler)
		defer server.Close()
		s := poolService.(*Service)
		s.taskService = mockTask

		gotID, err := s.createResource(context.Background(), poolID, "vm", nil)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, gotID)
	})

	t.Run("task failed status", func(t *testing.T) {
		poolID := uuid.Must(uuid.NewV4())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockTask := mock.NewMockTask(ctrl)
		mockTask.EXPECT().HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: "task-response"}, true).
			Return(&payloads.Task{
				Status: payloads.Failure,
				Result: payloads.Result{Message: "creation failed"},
			}, nil)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("{\"taskId\":\"task-response\"}"))
		})
		poolService, server := setupTestServer(t, handler)
		defer server.Close()
		s := poolService.(*Service)
		s.taskService = mockTask

		gotID, err := s.createResource(context.Background(), poolID, "vm", nil)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, gotID)
	})
}

func TestCreateNetworkParams(t *testing.T) {
	t.Run("validation: empty name", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("should not call API when validation fails")
		})
		poolService, server := setupTestServer(t, handler)
		defer server.Close()

		s := poolService
		// Call CreateNetwork with empty name
		gotID, err := s.CreateNetwork(context.Background(), uuid.Must(uuid.NewV4()), payloads.CreateNetworkParams{
			Name: "",
			Vlan: 100,
			Pif:  uuid.Must(uuid.NewV4()),
		})
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, gotID)
	})

	t.Run("validation: vlan out of range", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("should not call API when validation fails")
		})
		poolService, server := setupTestServer(t, handler)
		defer server.Close()

		s := poolService
		gotID, err := s.CreateNetwork(context.Background(), uuid.Must(uuid.NewV4()), payloads.CreateNetworkParams{
			Name: "net",
			Vlan: 5000,
			Pif:  uuid.Must(uuid.NewV4()),
		})
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, gotID)
	})

	t.Run("validation: nil pifId", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("should not call API when validation fails")
		})
		poolService, server := setupTestServer(t, handler)
		defer server.Close()

		s := poolService
		gotID, err := s.CreateNetwork(context.Background(), uuid.Must(uuid.NewV4()), payloads.CreateNetworkParams{
			Name: "net",
			Vlan: 100,
			// PifID: uuid.Nil, // omitted to test zero value
		})
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, gotID)
	})

	t.Run("forwards params in POST body", func(t *testing.T) {
		poolID := uuid.Must(uuid.NewV4())
		expectedID := uuid.Must(uuid.NewV4())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockTask := mock.NewMockTask(ctrl)
		mockTask.EXPECT().HandleTaskResponse(gomock.Any(), gomock.Any(), true).Return(&payloads.Task{
			Status: payloads.Success,
			Result: payloads.Result{ID: expectedID},
		}, nil)

		// handler will verify the JSON body contains the fields
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body payloads.CreateNetworkParams
			err := json.NewDecoder(r.Body).Decode(&body)
			assert.NoError(t, err)
			assert.Equal(t, "mynet", body.Name)
			assert.Equal(t, uint(1500), *body.MTU)
			assert.Equal(t, uint(100), body.Vlan)
			// reply with a task response string
			_, _ = w.Write([]byte("{\"taskId\":\"task-response\"}"))
		})

		poolService, server := setupTestServer(t, handler)
		defer server.Close()
		// Use createResource directly so we can pass pointer params and ensure body encoding
		s := poolService.(*Service)
		s.taskService = mockTask

		params := payloads.CreateNetworkParams{
			Name: "mynet",
			MTU:  func() *uint { v := uint(1500); return &v }(),
			Vlan: 100,
			Pif:  uuid.Must(uuid.NewV4()),
		}
		gotID, err := s.CreateNetwork(context.Background(), poolID, params)
		assert.NoError(t, err)
		assert.Equal(t, expectedID, gotID)
	})
}

func TestPerformPoolAction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		poolID := uuid.Must(uuid.NewV4())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockTask := mock.NewMockTask(ctrl)
		mockTask.EXPECT().HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: "task-response"}, true).
			Return(&payloads.Task{
				Status: payloads.Success,
			}, nil)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte("{\"taskId\":\"task-response\"}"))
		})

		poolService, server := setupTestServer(t, handler)
		defer server.Close()
		s := poolService.(*Service)
		s.taskService = mockTask

		err := s.performPoolAction(context.Background(), poolID, "emergency_shutdown")
		assert.NoError(t, err)
	})

	t.Run("http error", func(t *testing.T) {
		poolID := uuid.Must(uuid.NewV4())
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "fail", http.StatusInternalServerError)
		})
		poolService, server := setupTestServer(t, handler)
		defer server.Close()
		s := poolService.(*Service)

		err := s.performPoolAction(context.Background(), poolID, "rolling_reboot")
		assert.Error(t, err)
	})

	t.Run("task handler error", func(t *testing.T) {
		poolID := uuid.Must(uuid.NewV4())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockTask := mock.NewMockTask(ctrl)
		mockTask.EXPECT().HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: "task-response"}, true).
			Return(nil, fmt.Errorf("boom"))

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("{\"taskId\":\"task-response\"}"))
		})
		poolService, server := setupTestServer(t, handler)
		defer server.Close()
		s := poolService.(*Service)
		s.taskService = mockTask

		err := s.performPoolAction(context.Background(), poolID, "rolling_update")
		assert.Error(t, err)
	})

	t.Run("task failed status", func(t *testing.T) {
		poolID := uuid.Must(uuid.NewV4())

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockTask := mock.NewMockTask(ctrl)
		mockTask.EXPECT().HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: "task-response"}, true).
			Return(&payloads.Task{
				Status: payloads.Failure,
				Result: payloads.Result{Message: "failed action"},
			}, nil)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("{\"taskId\":\"task-response\"}"))
		})
		poolService, server := setupTestServer(t, handler)
		defer server.Close()
		s := poolService.(*Service)
		s.taskService = mockTask

		err := s.performPoolAction(context.Background(), poolID, "emergency_shutdown")
		assert.Error(t, err)
	})
}
