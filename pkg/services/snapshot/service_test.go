package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	mock_library "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library/mock"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

const (
	errorID = "b509922c-6982-4711-86a4-cb6784c07468"
	// notFoundID is a UUID string used to trigger not found errors in tests
	notFoundID = "11111111-1111-1111-1111-111111111111"
)

func setupSnapshotTestServer(t *testing.T) (*httptest.Server, library.Snapshot) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasPrefix(r.URL.Path, "/rest/v0/") {
			switch {
			case strings.HasPrefix(r.URL.Path, "/rest/v0/vm-snapshots/") && r.Method == http.MethodGet:
				parts := strings.Split(r.URL.Path, "/")
				snapshotIDStr := parts[len(parts)-1]
				id, err := uuid.FromString(snapshotIDStr)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid snapshot UUID format"})
					return
				}

				if id == uuid.Must(uuid.FromString(notFoundID)) {
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "snapshot not found"})
					return
				}

				vmID := uuid.Must(uuid.NewV4())
				snapshot := payloads.Snapshot{
					ID:              id,
					NameLabel:       "test-snapshot",
					NameDescription: "Test snapshot description",
					SnapshotOf:      vmID,
					SnapshotTime:    time.Now().Unix(),
				}

				if err := json.NewEncoder(w).Encode(snapshot); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Printf("Error encoding snapshot response: %v\n", err)
				}
				return

			case r.URL.Path == "/rest/v0/vm-snapshots" && r.Method == http.MethodGet:
				var snapshotURLs []string

				snapshot1ID := uuid.Must(uuid.NewV4())
				snapshot2ID := uuid.Must(uuid.NewV4())

				snapshotURLs = append(snapshotURLs,
					fmt.Sprintf("/rest/v0/vm-snapshots/%s", snapshot1ID),
					fmt.Sprintf("/rest/v0/vm-snapshots/%s", snapshot2ID),
				)

				if err := json.NewEncoder(w).Encode(snapshotURLs); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

			case strings.HasPrefix(r.URL.Path, "/rest/v0/vm-snapshots/") && r.Method == http.MethodDelete:
				parts := strings.Split(r.URL.Path, "/")
				snapshotID := parts[len(parts)-1]

				if snapshotID == errorID {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				response := struct {
					Success bool `json:"success"`
				}{
					Success: true,
				}

				if err := json.NewEncoder(w).Encode(response); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

			case strings.HasSuffix(r.URL.Path, "/snapshot") && r.Method == http.MethodPost:
				var params struct {
					NameLabel string `json:"name_label"`
				}

				if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				parts := strings.Split(r.URL.Path, "/")
				vmIDStr := parts[len(parts)-3]
				vmID, err := uuid.FromString(vmIDStr)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				snapshotID := uuid.Must(uuid.NewV4())
				if params.NameLabel == "trigger-task" {
					taskURL := fmt.Sprintf("/rest/v0/tasks/%s", uuid.Must(uuid.NewV4()))
					fmt.Fprint(w, taskURL)
					return
				}

				snapshot := payloads.Snapshot{
					ID:              snapshotID,
					NameLabel:       params.NameLabel,
					NameDescription: "Created via API",
					SnapshotOf:      vmID,
					SnapshotTime:    time.Now().Unix(),
				}

				if err := json.NewEncoder(w).Encode(snapshot); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

			default:
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}

		if r.URL.Path == "/" && r.Method == http.MethodPost {
			var request struct {
				Method string          `json:"method"`
				Params json.RawMessage `json:"params"`
				ID     int             `json:"id"`
			}

			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			switch request.Method {
			case "vm.revert":
				var params struct {
					VM       string `json:"vm"`
					Snapshot string `json:"snapshot"`
				}

				if err := json.Unmarshal(request.Params, &params); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				if params.Snapshot == errorID {
					response := map[string]any{
						"error": map[string]any{
							"code":    500,
							"message": "Error reverting snapshot",
						},
						"id": request.ID,
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					return
				}

				response := map[string]any{
					"result": true,
					"id":     request.ID,
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				return
			default:
				response := map[string]any{
					"error": map[string]any{
						"code":    404,
						"message": fmt.Sprintf("Method not found: %s", request.Method),
					},
					"id": request.ID,
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	serverURL := server.URL

	restClient := &client.Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: "http", Host: serverURL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
	}

	legacyClient := &v1.Client{
		// We only need this to satisfy the interface requirement
		// but it won't be used in the tests we're performing
	}

	ctrl := gomock.NewController(t)
	jsonrpcSvc := mock_library.NewMockJSONRPC(ctrl)

	jsonrpcSvc.EXPECT().
		Call("vm.revert", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result any, logContext ...zap.Field) error {
			snapshot, ok := params["snapshot"].(string)
			if ok && snapshot == errorID {
				return fmt.Errorf("error reverting snapshot")
			}

			*(result.(*bool)) = true
			return nil
		}).AnyTimes()

	jsonrpcSvc.EXPECT().
		ValidateResult(true, "snapshot revert", gomock.Any()).
		Return(nil).AnyTimes()

	jsonrpcSvc.EXPECT().
		ValidateResult(false, "snapshot revert", gomock.Any()).
		Return(fmt.Errorf("snapshot revert operation returned unsuccessful status")).AnyTimes()

	log, _ := logger.New(false)

	snapshotService := New(restClient, legacyClient, jsonrpcSvc, log)

	return server, snapshotService
}

func TestGetByID(t *testing.T) {
	server, service := setupSnapshotTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		id := uuid.Must(uuid.NewV4())
		snapshot, err := service.GetByID(ctx, id)

		assert.NoError(t, err)
		assert.NotNil(t, snapshot)
		assert.Equal(t, id, snapshot.ID)
		assert.Equal(t, "test-snapshot", snapshot.NameLabel)
	})

	t.Run("nonexistent snapshot", func(t *testing.T) {
		id := uuid.Must(uuid.FromString(notFoundID))
		snapshot, err := service.GetByID(ctx, id)

		assert.Error(t, err)
		assert.Nil(t, snapshot)
	})
}

func TestCreate(t *testing.T) {
	server, service := setupSnapshotTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		vmID := uuid.Must(uuid.NewV4())
		name := "test-create-snapshot"

		taskID, err := service.Create(ctx, vmID, name)

		assert.NoError(t, err)
		assert.NotEmpty(t, taskID)
	})
}

func TestDelete(t *testing.T) {
	server, service := setupSnapshotTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		id := uuid.Must(uuid.NewV4())
		err := service.Delete(ctx, id)

		assert.NoError(t, err)
	})

	t.Run("error delete", func(t *testing.T) {
		id, _ := uuid.FromString(errorID)
		err := service.Delete(ctx, id)

		assert.Error(t, err)
	})
}

func TestRevert(t *testing.T) {
	server, service := setupSnapshotTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("successful revert", func(t *testing.T) {
		vmID := uuid.Must(uuid.NewV4())
		snapshotID := uuid.Must(uuid.NewV4())

		err := service.Revert(ctx, vmID, snapshotID)

		assert.NoError(t, err)
	})

	t.Run("error revert", func(t *testing.T) {
		vmID := uuid.Must(uuid.NewV4())
		snapshotID, _ := uuid.FromString(errorID)

		err := service.Revert(ctx, vmID, snapshotID)

		assert.Error(t, err)
	})
}
