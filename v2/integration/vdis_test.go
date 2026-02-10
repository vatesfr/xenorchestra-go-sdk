package integration

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/docker/go-units"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
)

func TestVDIGet(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vdiTestID1 := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vdi1", 512*units.MB)
	vdiTestID2 := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vdi2", 512*units.MB)

	t.Run("GetSingleVDI", func(t *testing.T) {
		t.Parallel()
		vdi, err := client.VDI().Get(ctx, vdiTestID1)
		require.NoErrorf(t, err, "failed to get VDI %s", vdiTestID1)
		assert.Equal(t, vdiTestID1, vdi.ID, "VDI ID should match")
		assert.Equal(t, testPrefix+"vdi1", vdi.NameLabel, "VDI name label should match")
		assert.NotEmpty(t, vdi.NameLabel, "VDI name label should be populated")
	})

	t.Run("GetAllVDIsWithNoLimit", func(t *testing.T) {
		t.Parallel()
		vdis, err := client.VDI().GetAll(ctx, 0, "")
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(vdis), 2, "GetAll should return at least the two created VDIs")
	})

	t.Run("GetAllVDIsWithLimit", func(t *testing.T) {
		t.Parallel()
		vdis, err := client.VDI().GetAll(ctx, 1, "")
		require.NoError(t, err)
		require.Len(t, vdis, 1, "GetAll with limit=1 should return exactly one VDI")
	})

	t.Run("GetAllVDIsWithFilter", func(t *testing.T) {
		t.Parallel()
		vdis, err := client.VDI().GetAll(ctx, 0, "name_label:"+testPrefix+"vdi2")
		require.NoError(t, err)
		require.Len(t, vdis, 1, "GetAll with filter should return exactly one matching VDI")
		assert.Equal(t, vdiTestID2, vdis[0].ID, "Filtered VDI ID should match the expected VDI")
	})
	t.Run("GetInvalidID", func(t *testing.T) {
		t.Parallel()
		_, err := client.VDI().Get(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
		require.Error(t, err, "expected error when fetching VDI with invalid ID")
	})
}

func vdiTagExists(ctx context.Context, client library.Library, vdiID uuid.UUID, tag string) bool {
	vdi, err := client.VDI().Get(ctx, vdiID)
	if err != nil {
		return false
	}
	return slices.Contains(vdi.Tags, tag)
}

func TestVDITags(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vdiTestID := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vdi-tags", 512*units.MB)

	t.Run("AddTag", func(t *testing.T) {
		t.Parallel()
		tag := testPrefix + "tag"
		require.NoError(t, client.VDI().AddTag(ctx, vdiTestID, tag), "adding tag should succeed")
		require.Eventually(t, func() bool {
			return vdiTagExists(ctx, client, vdiTestID, tag)
		}, 1*time.Minute, 2*time.Second, "tag should be attached to the VDI")
	})

	t.Run("RemoveTag", func(t *testing.T) {
		t.Parallel()
		tag := testPrefix + "remove-tag"
		require.NoError(t, client.VDI().AddTag(ctx, vdiTestID, tag), "setup tag addition should succeed")
		require.Eventually(t, func() bool {
			return vdiTagExists(ctx, client, vdiTestID, tag)
		}, 1*time.Minute, 2*time.Second, "tag should be attached to the VDI")

		require.NoError(t, client.VDI().RemoveTag(ctx, vdiTestID, tag), "removing tag should succeed")
		require.Eventually(t, func() bool {
			return !vdiTagExists(ctx, client, vdiTestID, tag)
		}, 1*time.Minute, 2*time.Second, "tag should be removed from the VDI")
	})
}

func TestVDIDeletion(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vdiTestID := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vdi-delete", 512*units.MB)
	err := client.VDI().Delete(ctx, vdiTestID)
	require.NoError(t, err, "deleting VDI should succeed")

	_, err = client.VDI().Get(ctx, vdiTestID)
	assert.Error(t, err, "expected error when fetching deleted VDI")
}

func TestVDIMigration(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vdiTestID := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vdi-migrate", 512*units.MB)
	// We will migrate the VDI to the same SR, this is just to test the migration functionality without
	// needing to create a new SR for the test.
	srTestID := uuid.Must(uuid.FromString(intTests.testSR.Id))

	taskID, err := client.VDI().Migrate(ctx, vdiTestID, srTestID)
	require.NoError(t, err, "migrating VDI should succeed")
	require.NotEmpty(t, taskID, "migration should return a task ID")

	// Wait for the migration task to complete
	task, err := client.Task().Wait(ctx, taskID)
	require.NoError(t, err, "migration task should complete successfully")
	assert.NotNil(t, task, "migration task result should not be nil")

	// The completed task should indicate the new VDI ID in its result.
	// We can verify that the new VDI exists and has the expected SR ID and the VDI name.
	assert.NotEqual(t, uuid.Nil, task.Result.ID, "new VDI ID should be present in task result")
	newVDI, err := client.VDI().Get(ctx, task.Result.ID)
	require.NoError(t, err, "should be able to get the new VDI after migration")
	assert.Equal(t, srTestID, newVDI.SR, "new VDI should be in the target SR")
	assert.Equal(t, testPrefix+"vdi-migrate", newVDI.NameLabel, "new VDI should have the same name label as original VDI")

	// After migration, the VDI should have a new ID. We can check that the original VDI no longer exists
	// NOTE: as we are migrating on the same SR, the VDI will keep its ID, so this test is not relevant in this case.
	// require.Eventually(t, func() bool {
	// 	_, err := client.VDI().Get(ctx, vdiTestID)
	// 	return err != nil
	// }, 1*time.Minute, 2*time.Second, "original VDI should be deleted after migration")

}
