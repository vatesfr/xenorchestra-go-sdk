package jsonrpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
)

func setupJSONRPCTestServer() (*httptest.Server, library.JSONRPC) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			ID     int             `json:"id"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
			return
		}

		switch request.Method {
		case "success.method":
			response := map[string]interface{}{
				"result": "success-result",
				"id":     request.ID,
			}
			json.NewEncoder(w).Encode(response)

		case "error.method":
			response := map[string]interface{}{
				"error": map[string]interface{}{
					"code":    500,
					"message": "Internal server error",
				},
				"id": request.ID,
			}
			json.NewEncoder(w).Encode(response)

		case "boolean.true":
			response := map[string]interface{}{
				"result": true,
				"id":     request.ID,
			}
			json.NewEncoder(w).Encode(response)

		case "boolean.false":
			response := map[string]interface{}{
				"result": false,
				"id":     request.ID,
			}
			json.NewEncoder(w).Encode(response)

		case "complex.result":
			response := map[string]interface{}{
				"result": map[string]interface{}{
					"id":   "12345",
					"name": "Test Resource",
					"data": []interface{}{1, 2, 3},
				},
				"id": request.ID,
			}
			json.NewEncoder(w).Encode(response)

		default:
			response := map[string]interface{}{
				"error": map[string]interface{}{
					"code":    404,
					"message": fmt.Sprintf("Method not found: %s", request.Method),
				},
				"id": request.ID,
			}
			json.NewEncoder(w).Encode(response)
		}
	}))

	client, err := v1.NewClient(v1.Config{Url: server.URL})
	if err != nil {
		panic(fmt.Sprintf("Failed to create client: %v", err))
	}

	log, err := logger.New(false)
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}

	return server, New(client.(*v1.Client), log)
}

func TestCall(t *testing.T) {
	server, jsonrpcSvc := setupJSONRPCTestServer()
	defer server.Close()

	t.Run("successful call", func(t *testing.T) {
		var result string
		err := jsonrpcSvc.Call("success.method", map[string]any{
			"param1": "value1",
			"param2": 123,
		}, &result)

		assert.NoError(t, err)
		assert.Equal(t, "success-result", result)
	})

	t.Run("error call", func(t *testing.T) {
		var result string
		err := jsonrpcSvc.Call("error.method", map[string]any{
			"param1": "value1",
		}, &result)

		assert.Error(t, err)
		assert.Empty(t, result)
	})

	t.Run("method not found", func(t *testing.T) {
		var result string
		err := jsonrpcSvc.Call("not.found", map[string]any{}, &result)

		assert.Error(t, err)
		assert.Empty(t, result)
	})

	t.Run("complex result", func(t *testing.T) {
		var result map[string]interface{}
		err := jsonrpcSvc.Call("complex.result", map[string]any{}, &result)

		assert.NoError(t, err)
		assert.Equal(t, "12345", result["id"])
		assert.Equal(t, "Test Resource", result["name"])
		assert.NotNil(t, result["data"])
	})

	t.Run("with log context", func(t *testing.T) {
		var result string
		err := jsonrpcSvc.Call("success.method", map[string]any{
			"param1": "value1",
		}, &result, zap.String("context", "test-context"))

		assert.NoError(t, err)
		assert.Equal(t, "success-result", result)
	})
}

func TestValidateResult(t *testing.T) {
	server, jsonrpcSvc := setupJSONRPCTestServer()
	defer server.Close()

	t.Run("true result", func(t *testing.T) {
		var result bool
		err := jsonrpcSvc.Call("boolean.true", map[string]any{}, &result)
		assert.NoError(t, err)
		assert.True(t, result)

		err = jsonrpcSvc.ValidateResult(result, "test operation")
		assert.NoError(t, err)
	})

	t.Run("false result", func(t *testing.T) {
		var result bool
		err := jsonrpcSvc.Call("boolean.false", map[string]any{}, &result)
		assert.NoError(t, err)
		assert.False(t, result)

		err = jsonrpcSvc.ValidateResult(result, "test operation")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test operation returned unsuccessful status")
	})

	t.Run("with log context", func(t *testing.T) {
		err := jsonrpcSvc.ValidateResult(false, "test operation", zap.String("resource", "test-resource"))
		assert.Error(t, err)
	})
}
