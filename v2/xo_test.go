package v2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
)

func TestNewClientWithoutV1Connection(t *testing.T) {
	// Test that v2 client can be created without requiring a valid v1 XOA connection.
	// The v1 client should only be initialized on first use.
	// We use a token to avoid authentication at creation time.

	cfg := &config.Config{
		Url:   "http://localhost:9999",
		Token: "test-token",
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
		Url:   "http://localhost:9999",
		Token: "test-token",
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
