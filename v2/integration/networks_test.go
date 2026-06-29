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
	t.Parallel()
	ctx, client, prefix := SetupTestContext(t)

	networkID1 := createNetworkForTest(t, ctx, client, prefix+"test-network-1")
	createNetworkForTest(t, ctx, client, prefix+"test-network-2")

	t.Run("NoLimit", func(t *testing.T) {
		t.Parallel()
		networks, err := client.Network().GetAll(ctx, 0, "")
		require.NoError(t, err)
		require.NotNil(t, networks)
		assert.GreaterOrEqual(t, len(networks), 2, "GetAll should return at least two networks in the test environment")

		network := networks[0]
		assert.NotEqual(t, uuid.Nil, network.ID, "network ID should be set")
		assert.NotEmpty(t, network.NameLabel, "network name label should be set")
		assert.NotEmpty(t, network.Bridge, "network bridge should be set")
		assert.Greater(t, network.MTU, 0, "network MTU should be positive")
	})

	t.Run("WithLimit", func(t *testing.T) {
		t.Parallel()
		networks, err := client.Network().GetAll(ctx, 1, "")
		require.NoError(t, err)
		assert.Len(t, networks, 1, "GetAll with limit=1 should return exactly one network")
	})

	t.Run("WithFilter", func(t *testing.T) {
		t.Parallel()
		filter := "name_label:" + prefix + "test-network-1"
		networks, err := client.Network().GetAll(ctx, 0, filter)
		require.NoError(t, err)
		require.NotNil(t, networks)
		assert.Len(t, networks, 1, "GetAll with filter should return exactly one network")
		assert.Equal(t, networkID1, networks[0].ID, "filtered network ID should match the expected ID")
	})
}

func TestNetworkGet(t *testing.T) {
	ctx, client, prefix := SetupTestContext(t)

	networkID := createNetworkForTest(t, ctx, client, prefix+"test-network-1")

	t.Run("with valid ID", func(t *testing.T) {
		t.Parallel()
		network, err := client.Network().Get(ctx, networkID)
		require.NoErrorf(t, err, "failed to get network %s", networkID)
		assert.Equal(t, networkID, network.ID, "network ID should match")
		assert.NotEmpty(t, network.NameLabel, "network name label should be populated")
		assert.NotEmpty(t, network.Bridge, "network bridge should be populated")
	})

	t.Run("with invalid ID", func(t *testing.T) {
		_, err := client.Network().Get(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
		require.Error(t, err, "expected error when fetching network with invalid ID")
	})

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

	networkID := createNetworkForTest(t, ctx, client, prefix+"test-network-1")
	tag := prefix + "tag"

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

	networkID := createNetworkForTest(t, ctx, client, prefix+"test-network-1")

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

func TestNetworkGetTasks(t *testing.T) {
	ctx, client, prefix := SetupTestContext(t)

	networkID := createNetworkForTest(t, ctx, client, prefix+"test-network-1")

	t.Run("with valid network ID", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.Network().GetTasks(ctx, networkID, 0, "")
		require.NoError(t, err)
		require.NotNil(t, tasks)
		assert.GreaterOrEqual(t, len(tasks), 0, "GetTasks should return zero or more tasks for the network")
	})

	t.Run("with invalid network ID", func(t *testing.T) {
		t.Parallel()
		_, err := client.Network().GetTasks(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"), 0, "")
		require.Error(t, err, "expected error when fetching tasks with invalid network ID")
	})
}

func TestNetworkDelete(t *testing.T) {
	ctx, client, prefix := SetupTestContext(t)

	networkID := createNetworkForTest(t, ctx, client, prefix+"test-network-to-delete")

	t.Run("with valid network ID", func(t *testing.T) {
		t.Parallel()
		err := client.Network().Delete(ctx, networkID)
		require.NoError(t, err, "deleting network should succeed")

		// Verify that the network no longer exists
		_, err = client.Network().Get(ctx, networkID)
		require.Error(t, err, "expected error when fetching deleted network")
	})

	t.Run("with invalid network ID", func(t *testing.T) {
		t.Parallel()
		err := client.Network().Delete(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
		require.Error(t, err, "expected error when deleting network with invalid ID")
	})
}
