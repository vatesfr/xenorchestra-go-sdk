package v2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/sourcegraph/jsonrpc2/websocket"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
)

const (
	unreachableURL = "http://localhost:9999"
	testToken      = "test-token"
)

func TestNewClientWithoutV1Connection(t *testing.T) {
	// Test that v2 client can be created without requiring a valid v1 XOA connection.
	// The v1 client should only be initialized on first use.
	// We use a token to avoid authentication at creation time.

	cfg := &config.Config{
		Url:   unreachableURL,
		Token: testToken,
	}

	// This should succeed even though the URL is unreachable
	// (Token prevents auth attempt)
	client, err := New(cfg)
	assert.NoError(t, err, "v2 client creation should not fail without reachable XOA")
	assert.NotNil(t, client)

	// Verify that the basic services are available
	assert.NotNil(t, client.VM())
	assert.NotNil(t, client.Task())
	assert.NotNil(t, client.Pool())
}

func TestV1ClientLazyInitialization(t *testing.T) {
	// Test that v1 client is initialized lazily on first access.

	cfg := &config.Config{
		Url:   unreachableURL,
		Token: testToken,
	}

	client, err := New(cfg)
	require.NoError(t, err)

	xoClient, ok := client.(*XOClient)
	require.True(t, ok, "client should be *XOClient")

	// At this point, v1Client should be nil
	assert.Nil(t, xoClient.v1Client, "v1Client should not be initialized yet")

	// Accessing V1Client() will trigger lazy initialization
	// This will fail because XOA is not reachable, but it demonstrates the lazy pattern
	v1Client := client.V1Client()
	// The call will return nil due to initialization error, but no panic should occur
	// In a real scenario with a valid XOA, this would return the initialized client
	assert.Nil(t, v1Client, "v1Client should be nil due to connection error")
	assert.NotNil(t, xoClient.v1InitErr, "v1Err should contain the connection error")
}

func TestCloseWithoutV1ClientPreventsLazyInitialization(t *testing.T) {
	cfg := &config.Config{
		Url:   unreachableURL,
		Token: testToken,
	}

	client, err := New(cfg)
	require.NoError(t, err)

	require.NoError(t, client.Close())
	require.NoError(t, client.Close(), "Close should be idempotent")

	xoClient := client.(*XOClient)
	assert.Nil(t, client.V1Client(), "a closed client must not open a v1 connection")
	assert.ErrorIs(t, xoClient.v1InitErr, errClientClosed)
}

func TestCloseClosesInitializedV1Client(t *testing.T) {
	disconnected := make(chan struct{}, 1)
	upgrader := gorillawebsocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		conn := jsonrpc2.NewConn(context.Background(), websocket.NewObjectStream(ws),
			jsonrpc2.HandlerWithError(func(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (any, error) {
				if req.Method == "session.signIn" {
					return map[string]string{"id": "user-id"}, nil
				}
				return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: req.Method}
			}))
		<-conn.DisconnectNotify()
		disconnected <- struct{}{}
	}))
	defer server.Close()

	client, err := New(&config.Config{
		Url:   server.URL,
		Token: testToken,
	})
	require.NoError(t, err)
	require.NotNil(t, client.V1Client(), "v1 client should connect to the fake server")

	require.NoError(t, client.Close())

	select {
	case <-disconnected:
	case <-time.After(5 * time.Second):
		t.Fatal("v1 websocket connection was not closed")
	}
	assert.Nil(t, client.V1Client(), "V1Client should return nil once closed")
}
