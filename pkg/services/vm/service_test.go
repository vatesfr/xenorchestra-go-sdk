package vm

import (
	"context"
	"encoding/json"
	"log/slog"
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

func setupTestServer(t *testing.T) (*httptest.Server, library.VM, *mock.MockPool) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var vmID uuid.UUID
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) >= 3 && pathParts[1] == "rest" && pathParts[2] == "v0" {
			if len(pathParts) >= 5 && pathParts[3] == "vms" {
				idStr := pathParts[4]
				if id, err := uuid.FromString(idStr); err == nil {
					vmID = id
				}
			}
		}
		t.Logf("Request made to mock server: %s %s", r.Method, r.URL.Path)
		t.Logf("VM ID parsed: %s", vmID.String())

		switch {
		// List VMs
		case r.URL.Path == "/rest/v0/vms" && r.Method == http.MethodGet:
			err := json.NewEncoder(w).Encode([]payloads.VM{
				{
					ID:         uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000001")),
					NameLabel:  "VM 1",
					PowerState: payloads.PowerStateRunning,
				},
				{
					ID:         uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000002")),
					NameLabel:  "VM 2",
					PowerState: payloads.PowerStateHalted,
				},
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		// Get VM by ID
		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && len(pathParts) == 5 && r.Method == http.MethodGet:
			vm := payloads.VM{
				ID:         vmID,
				NameLabel:  "",
				PowerState: payloads.PowerStateRunning,
			}
			switch vmID.String() {
			case "00000000-0000-0000-0000-000000000001":
				vm.NameLabel = "VM 1"
			case "00000000-0000-0000-0000-000000000002":
				vm.NameLabel = "VM 2"
			case "10000000-0000-0000-0000-000000000001":
				vm.NameLabel = "New VM"
				vm.PowerState = payloads.PowerStateHalted
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}

			err := json.NewEncoder(w).Encode(vm)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		// Delete VM by ID
		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && len(pathParts) == 5 && r.Method == http.MethodDelete:
			err := json.NewEncoder(w).Encode(map[string]bool{"success": true})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		// Actions on VM
		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && len(pathParts) == 7 && r.Method == http.MethodPost:
			action := strings.Split(r.URL.Path, "/")[6]
			t.Logf("Action requested: %s", action)
			switch action {
			case "start", "clean_shutdown", "hard_shutdown", "clean_reboot", "hard_reboot", "snapshot", "restart", "suspend", "resume", "pause", "unpause":
				err := json.NewEncoder(w).Encode(payloads.TaskIDResponse{TaskID: "task-123"})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		default:
			slog.Warn("Unhandled path", "path", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	restClient := &client.Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
	}

	log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
	if err != nil {
		panic(err)
	}

	// Create mock controller and mocks
	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)
	mockPool := mock.NewMockPool(ctrl)

	return server, New(restClient, mockTask, mockPool, log), mockPool
}

func TestGetByID(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	id := uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
	vm, err := service.GetByID(context.Background(), id)

	assert.NoError(t, err)
	assert.Equal(t, id, vm.ID)
	assert.Equal(t, "VM 1", vm.NameLabel)
	assert.Equal(t, payloads.PowerStateRunning, vm.PowerState)
}

func TestGetAll(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	vms, err := service.GetAll(context.Background(), 0, "")

	assert.NoError(t, err)
	assert.Len(t, vms, 2)
	assert.Equal(t, "VM 1", vms[0].NameLabel)
	assert.Equal(t, "VM 2", vms[1].NameLabel)
}

func TestCreate(t *testing.T) {
	server, service, mockPool := setupTestServer(t)
	defer server.Close()

	// Set up mock expectations for Pool.CreateVM
	poolID := uuid.FromStringOrNil("201b228b-2f91-4138-969c-49cae8780448")
	vmID := uuid.Must(uuid.FromString("10000000-0000-0000-0000-000000000001"))

	createParams := payloads.CreateVMParams{
		NameLabel:       "New VM",
		NameDescription: "Test VM",
	}

	mockPool.EXPECT().CreateVM(gomock.Any(), poolID, createParams).Return(vmID, nil)

	vm, err := service.Create(context.Background(), poolID, &createParams)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, vm.ID)
	assert.Equal(t, "New VM", vm.NameLabel)
	assert.Equal(t, payloads.PowerStateHalted, vm.PowerState)
}

// TODO: Re-enable when Update is implemented
// func TestUpdate(t *testing.T) {
// 	server, service, _ := setupTestServer(t)
// 	defer server.Close()

// 	id := uuid.Must(uuid.NewV4())
// 	updateVM := &payloads.VM{
// 		ID:              id,
// 		NameLabel:       "Updated VM",
// 		NameDescription: "Updated description",
// 	}

// 	vm, err := service.Update(context.Background(), updateVM)

// 	assert.NoError(t, err)
// 	assert.Equal(t, id, vm.ID)
// 	assert.Equal(t, "Updated VM", vm.NameLabel)
// 	assert.Equal(t, "Updated description", vm.NameDescription)
// }

func TestDelete(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	id := uuid.Must(uuid.NewV4())
	err := service.Delete(context.Background(), id)

	assert.NoError(t, err)
}

func TestPowerOperations(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	// Get access to the mockTask from service
	s := service.(*Service)
	mockTask := s.taskService.(*mock.MockTask)

	id := uuid.Must(uuid.NewV4())

	// Expect calls to HandleTaskResponse
	mockTask.EXPECT().HandleTaskResponse(gomock.Any(), gomock.Any(), false).Return(&payloads.Task{ID: "task-123"}, nil).AnyTimes()

	taskID, err := service.Start(context.Background(), id, nil)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)

	taskID, err = service.CleanShutdown(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)

	taskID, err = service.Suspend(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)

	taskID, err = service.Resume(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)

	taskID, err = service.Pause(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)

	taskID, err = service.Unpause(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)
}
