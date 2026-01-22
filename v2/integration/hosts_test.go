package integration

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostGetAll(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	hosts, err := client.Host().GetAll(ctx, 0, "")
	require.NoError(t, err)
	require.NotEmpty(t, hosts, "GetAll should return at least one host in the test environment")

	// Basic sanity checks on the first host
	host := hosts[0]
	assert.NotEqual(t, uuid.Nil, host.ID, "host ID should be set")
	assert.NotEmpty(t, host.NameLabel, "host name label should be set")
	assert.NotEmpty(t, host.ProductBrand, "host productBrand should be set")
}

func TestHostGet(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	// Reuse GetAll to discover an existing host
	hosts, err := client.Host().GetAll(ctx, 1, "")
	require.NoError(t, err)
	require.NotEmpty(t, hosts, "expected at least one host")

	hostID := hosts[0].ID
	host, err := client.Host().Get(ctx, hostID)
	require.NoErrorf(t, err, "failed to get host %s", hostID)
	assert.Equal(t, hostID, host.ID, "host ID should match")
	assert.NotEmpty(t, host.NameLabel, "host name label should be populated")
}

func TestHostGetInvalidID(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	_, err := client.Host().Get(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
	require.Error(t, err, "expected error when fetching host with invalid ID")
}
