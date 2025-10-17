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

func setupTestServer(t *testing.T) (*httptest.Server, library.VM, *mock.MockTask) {
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

		switch {
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

		case r.URL.Path == "/rest/v0/vms" && r.Method == http.MethodPost:
			var vm payloads.VM
			_ = json.NewDecoder(r.Body).Decode(&vm)
			vm.ID = uuid.Must(uuid.NewV4())
			vm.PowerState = payloads.PowerStateHalted
			_ = json.NewEncoder(w).Encode(vm)

		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && len(pathParts) == 5 && r.Method == http.MethodGet:
			vmName := "VM 1"
			if vmID.String() == "00000000-0000-0000-0000-000000000002" {
				vmName = "VM 2"
			}

			err := json.NewEncoder(w).Encode(payloads.VM{
				ID:         vmID,
				NameLabel:  vmName,
				PowerState: payloads.PowerStateRunning,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && len(pathParts) == 5 && r.Method == http.MethodPost:
			var vm payloads.VM
			_ = json.NewDecoder(r.Body).Decode(&vm)
			vm.ID = vmID
			_ = json.NewEncoder(w).Encode(vm)

		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && len(pathParts) == 5 && r.Method == http.MethodDelete:
			err := json.NewEncoder(w).Encode(map[string]bool{"success": true})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/_/actions/"):
			action := strings.TrimPrefix(r.URL.Path, "/rest/v0/vms/_/actions/")

			var requestBody map[string]string
			_ = json.NewDecoder(r.Body).Decode(&requestBody)
			_, err := uuid.FromString(requestBody["id"])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			switch action {
			case "start", "clean_shutdown", "hard_shutdown", "clean_reboot", "hard_reboot", "snapshot":
				err := json.NewEncoder(w).Encode(map[string]bool{"success": true})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}

		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && strings.HasSuffix(r.URL.Path, "/suspend"):
			err := json.NewEncoder(w).Encode(map[string]bool{"success": true})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && strings.HasSuffix(r.URL.Path, "/resume"):
			err := json.NewEncoder(w).Encode(map[string]bool{"success": true})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case strings.HasPrefix(r.URL.Path, "/rest/v0/pools/") && strings.HasSuffix(r.URL.Path, "/actions/create_vm"):
			var createParams map[string]any
			_ = json.NewDecoder(r.Body).Decode(&createParams)

			pathParts := strings.Split(r.URL.Path, "/")
			poolID := pathParts[4]

			vm := payloads.VM{
				ID:              uuid.Must(uuid.NewV4()),
				NameLabel:       createParams["name_label"].(string),
				NameDescription: createParams["name_description"].(string),
				PowerState:      payloads.PowerStateHalted,
				PoolID:          uuid.Must(uuid.FromString(poolID)),
			}

			_ = json.NewEncoder(w).Encode(vm)

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

	log, err := logger.New(false)
	if err != nil {
		panic(err)
	}

	// Create mock controller and task mock
	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)

	return server, New(restClient, mockTask, log), mockTask
}

func TestGetByID(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	id := uuid.Must(uuid.NewV4())
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
	server, service, mockTask := setupTestServer(t)
	defer server.Close()

	// Set up mock expectations for task handling
	mockTask.EXPECT().HandleTaskResponse(gomock.Any(), gomock.Any(), true).Return(nil, false, nil).AnyTimes()

	newVM := &payloads.VM{
		NameLabel:       "New VM",
		NameDescription: "Test VM",
		CPUs: payloads.CPUs{
			Number: 2,
		},
		Memory: payloads.Memory{
			Static: []int64{1073741824, 4294967296},
		},
		PoolID: uuid.FromStringOrNil("201b228b-2f91-4138-969c-49cae8780448"),
	}

	vm, err := service.Create(context.Background(), newVM)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, vm.ID)
	assert.Equal(t, "New VM", vm.NameLabel)
	assert.Equal(t, payloads.PowerStateHalted, vm.PowerState)
}

func TestUpdate(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	id := uuid.Must(uuid.NewV4())
	updateVM := &payloads.VM{
		ID:              id,
		NameLabel:       "Updated VM",
		NameDescription: "Updated description",
	}

	vm, err := service.Update(context.Background(), updateVM)

	assert.NoError(t, err)
	assert.Equal(t, id, vm.ID)
	assert.Equal(t, "Updated VM", vm.NameLabel)
	assert.Equal(t, "Updated description", vm.NameDescription)
}

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

	id := uuid.Must(uuid.NewV4())

	err := service.Start(context.Background(), id)
	assert.NoError(t, err)

	err = service.CleanShutdown(context.Background(), id)
	assert.NoError(t, err)

	err = service.Suspend(context.Background(), id)
	assert.NoError(t, err)

	err = service.Resume(context.Background(), id)
	assert.NoError(t, err)
}
