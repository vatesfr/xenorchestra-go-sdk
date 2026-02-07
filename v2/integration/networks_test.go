package integration

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
)

func TestNetworkGetAll(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	networks, err := client.Network().GetAll(ctx, 0, "")
	require.NoError(t, err)
	require.NotEmpty(t, networks, "GetAll should return at least one network in the test environment")

	network := networks[0]
	assert.NotEqual(t, uuid.Nil, network.ID, "network ID should be set")
	assert.NotEmpty(t, network.NameLabel, "network name label should be set")
	assert.NotEmpty(t, network.Bridge, "network bridge should be set")
	assert.Greater(t, network.MTU, 0, "network MTU should be positive")
}

func TestNetworkGetAllWithLimit(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	networks, err := client.Network().GetAll(ctx, 1, "")
	require.NoError(t, err)
	assert.Len(t, networks, 1, "GetAll with limit=1 should return exactly one network")
}

func TestNetworkGet(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	// Discover an existing network via GetAll
	networks, err := client.Network().GetAll(ctx, 1, "")
	require.NoError(t, err)
	require.NotEmpty(t, networks, "expected at least one network")

	networkID := networks[0].ID
	network, err := client.Network().Get(ctx, networkID)
	require.NoErrorf(t, err, "failed to get network %s", networkID)
	assert.Equal(t, networkID, network.ID, "network ID should match")
	assert.NotEmpty(t, network.NameLabel, "network name label should be populated")
	assert.NotEmpty(t, network.Bridge, "network bridge should be populated")
}

func TestNetworkGetInvalidID(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	_, err := client.Network().Get(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
	require.Error(t, err, "expected error when fetching network with invalid ID")
}

func networkTagExists(ctx context.Context, client library.Library, networkID uuid.UUID, tag string) bool {
	network, err := client.Network().Get(ctx, networkID)
	if err != nil {
		return false
	}
	return slices.Contains(network.Tags, tag)
}

func TestNetworkAddTag(t *testing.T) {
	ctx, client, prefix := SetupTestContext(t)

	networks, err := client.Network().GetAll(ctx, 1, "")
	require.NoError(t, err)
	require.NotEmpty(t, networks, "expected at least one network")

	networkID := networks[0].ID
	tag := prefix + "tag"

	t.Cleanup(func() {
		_ = client.Network().RemoveTag(ctx, networkID, tag)
	})

	require.NoError(t, client.Network().AddTag(ctx, networkID, tag), "adding tag should succeed")

	require.Eventually(t, func() bool {
		return networkTagExists(ctx, client, networkID, tag)
	}, 1*time.Minute, 2*time.Second, "tag should be attached to the network")

	refreshed, err := client.Network().Get(ctx, networkID)
	require.NoError(t, err)
	assert.Contains(t, refreshed.Tags, tag, "network tags should contain the newly added tag")
}

func TestNetworkRemoveTag(t *testing.T) {
	ctx, client, prefix := SetupTestContext(t)

	networks, err := client.Network().GetAll(ctx, 1, "")
	require.NoError(t, err)
	require.NotEmpty(t, networks, "expected at least one network")

	networkID := networks[0].ID
	tag := prefix + "remove-tag"

	require.NoError(t, client.Network().AddTag(ctx, networkID, tag), "setup tag addition should succeed")

	require.Eventually(t, func() bool {
		return networkTagExists(ctx, client, networkID, tag)
	}, 1*time.Minute, 2*time.Second, "tag should be attached to the network")

	require.NoError(t, client.Network().RemoveTag(ctx, networkID, tag), "removing tag should succeed")

	require.Eventually(t, func() bool {
		return !networkTagExists(ctx, client, networkID, tag)
	}, 1*time.Minute, 2*time.Second, "tag should be removed from the network")

	refreshed, err := client.Network().Get(ctx, networkID)
	require.NoError(t, err)
	assert.NotContains(t, refreshed.Tags, tag, "network tags should not contain the removed tag")
}
