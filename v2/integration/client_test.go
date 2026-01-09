package integration

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	v2 "github.com/vatesfr/xenorchestra-go-sdk/v2"
)

// TestV1ClientLazyInitializationWithRealConnection tests that the v1 client is
// initialized lazily when accessed, and that it works with a real connection
func TestV1ClientLazyInitializationWithRealConnection(t *testing.T) {

	// Create a new client
	cfg, err := config.New()
	require.NoError(t, err, "failed to create config")

	client, err := v2.New(cfg)
	require.NoError(t, err, "failed to create client")

	// Use v2 services first (without touching v1 client)
	poolsV2, err := client.Pool().GetAll(integrationCtx, 1, "")
	require.NoError(t, err, "v2 services should work before v1 init")
	assert.NotEmpty(t, poolsV2)

	// At this point, v1Client should still not be initialized
	// Use reflection to check the v1Client field since it's unexported
	v1ClientField := reflect.ValueOf(client).Elem().FieldByName("v1Client")
	require.True(t, v1ClientField.IsValid(), "v1Client field should exist")

	// Check that v1Client is nil (zero value)
	assert.True(t, v1ClientField.IsNil(), "v1Client should not be initialized yet (should be nil)")

	// Access v1Client - this should trigger lazy initialization
	v1Client := client.V1Client()

	// With a real connection, v1Client should now be initialized
	// Test a simple v1 operation
	pools, err := v1Client.GetPoolByName(testPool.NameLabel)
	assert.NoError(t, err, "v1 client should work after initialization")
	assert.NotEmpty(t, pools, "should be able to fetch pools with v1 client")
}
