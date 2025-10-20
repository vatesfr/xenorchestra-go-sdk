package jsonrpc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"go.uber.org/zap"

	"github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	ws "github.com/sourcegraph/jsonrpc2/websocket"
	"github.com/stretchr/testify/assert"
)

const (
	fakeXoToken = "fake-token-123"
)

// testJSONRPCHandler implements a JSON-RPC handler for testing
type testJSONRPCHandler struct{}

func (h *testJSONRPCHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	switch req.Method {
	case "session.signIn":
		// Fake authentication success
		response := map[string]any{
			"user": map[string]any{
				"id":    "test-user",
				"email": "test@example.com",
			},
		}
		_ = conn.Reply(ctx, req.ID, response)

	case "success.method":
		_ = conn.Reply(ctx, req.ID, "success-result")

	case "error.method":
		_ = conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    500,
			Message: "Internal server error",
		})

	case "boolean.true":
		_ = conn.Reply(ctx, req.ID, true)

	case "boolean.false":
		_ = conn.Reply(ctx, req.ID, false)

	case "complex.result":
		response := map[string]any{
			"id":   "12345",
			"name": "Test Resource",
			"data": []any{1, 2, 3},
		}
		_ = conn.Reply(ctx, req.ID, response)

	default:
		_ = conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    404,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		})
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func setupJSONRPCTestServer() (*httptest.Server, library.JSONRPC) {
	// Create an HTTP server that upgrades to WebSocket
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api/") {
			// Upgrade the HTTP connection to WebSocket
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Create a WebSocket object stream and a JSON-RPC connection
			objStream := ws.NewObjectStream(conn)
			handler := &testJSONRPCHandler{}
			jsonrpcConn := jsonrpc2.NewConn(context.Background(), objStream, handler)

			// Wait for the connection to close
			<-jsonrpcConn.DisconnectNotify()
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	wsURL := strings.Replace(server.URL, "http", "ws", 1)

	client, err := v1.NewClient(v1.Config{
		Url:   wsURL,
		Token: fakeXoToken,
	})
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
		var result map[string]any
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
