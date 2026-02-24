package integration

import (
	"context"
	"io"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/docker/go-units"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
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

func TestVDIGetTasks(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	// Create and migrate the VDI multiple times to generate some tasks
	vdiTestID := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vdi-tasks", 512*units.MB)
	srTestID := uuid.Must(uuid.FromString(intTests.testSR.Id))
	taskID1, err := client.VDI().Migrate(ctx, vdiTestID, srTestID)
	require.NoError(t, err, "1st migrating VDI should succeed")
	task, err := client.Task().Wait(ctx, taskID1)
	require.NoError(t, err, "migration task should complete successfully")
	require.NotNil(t, task, "migration task result should not be nil")
	require.Equal(t, payloads.Success, task.Status, "migration task should complete successfully")
	taskID2, err := client.VDI().Migrate(ctx, vdiTestID, srTestID)
	require.NoError(t, err, "2nd migrating VDI should succeed")
	task, err = client.Task().Wait(ctx, taskID2)
	require.NoError(t, err, "migration task should complete successfully")
	require.NotNil(t, task, "migration task result should not be nil")
	require.Equal(t, payloads.Success, task.Status, "migration task should complete successfully")

	t.Run("GetTasksSuccess", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.VDI().GetTasks(ctx, vdiTestID, 0, "")
		require.NoError(t, err)
		require.NotNil(t, tasks)
		assert.GreaterOrEqual(t, len(tasks), 2, "there should be at least 2 tasks associated with the VDI")
	})

	t.Run("GetTasksWithLimit", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.VDI().GetTasks(ctx, vdiTestID, 1, "")
		require.NoError(t, err)
		require.NotNil(t, tasks)
		assert.Len(t, tasks, 1)
	})

	t.Run("GetTasksWithFilter", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.VDI().GetTasks(ctx, vdiTestID, 0, "status:failure")
		require.NoError(t, err)
		require.NotNil(t, tasks)
		assert.Len(t, tasks, 0)

		tasks, err = client.VDI().GetTasks(ctx, vdiTestID, 0, "id:"+taskID1)
		require.NoError(t, err)
		require.NotNil(t, tasks)
		assert.Len(t, tasks, 1)
	})

	t.Run("GetTasksInvalidVDI", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.VDI().GetTasks(ctx, uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440000")), 0, "")
		require.Error(t, err)
		assert.Nil(t, tasks)
	})
}

func TestVDIExport(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vdiTestID := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vdi-export-import", 10*units.MB)

	t.Run("export in raw", func(t *testing.T) {
		t.Parallel()

		err := client.VDI().Export(ctx, vdiTestID, payloads.VDIFormatRaw, func(reader io.Reader) error {
			// Verify the exported content is in raw format using qemu-img
			require.NotNil(t, reader, "exported content should not be nil")
			return verifyDiskFormat(t, reader, "raw")
		})
		require.NoError(t, err, "exporting VDI should succeed")
	})

	t.Run("export in vhd", func(t *testing.T) {
		t.Parallel()

		err := client.VDI().Export(ctx, vdiTestID, payloads.VDIFormatVHD, func(reader io.Reader) error {
			// Verify the exported content is in VHD format using qemu-img
			// Note: qemu-img identifies VHD format as "vpc" (Virtual PC)
			require.NotNil(t, reader, "exported content should not be nil")
			return verifyDiskFormat(t, reader, "vpc")
		})
		require.NoError(t, err, "exporting VDI should succeed")
	})

}

func TestVDIImportExport(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	t.Run("with raw disk", func(t *testing.T) {
		t.Parallel()

		// Create a VDI to import into
		vdiID := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vdi-import-raw", 10*units.MB)

		// Create a test RAW disk image
		diskPath := createTestDiskImage(t, "raw", 10*units.MB)
		defer os.Remove(diskPath)

		// Open the disk image file
		file, err := os.Open(diskPath)
		require.NoError(t, err, "opening test disk should succeed")
		defer file.Close()

		// Get file size
		fileInfo, err := file.Stat()
		require.NoError(t, err, "getting file info should succeed")

		// Import the disk into the VDI
		err = client.VDI().Import(ctx, vdiID, payloads.VDIFormatRaw, file, fileInfo.Size())
		require.NoError(t, err, "importing RAW disk should succeed")

		// Verify the VDI exists and has expected properties
		vdi, err := client.VDI().Get(ctx, vdiID)
		require.NoError(t, err, "getting imported VDI should succeed")
		assert.Equal(t, vdiID, vdi.ID, "VDI ID should match")
		assert.Equal(t, testPrefix+"vdi-import-raw", vdi.NameLabel, "VDI name should match")
		assert.Greater(t, vdi.Size, int64(0), "VDI should have non-zero size")
	})

	t.Run("with vhd disk", func(t *testing.T) {
		t.Parallel()

		// Create a VDI to import into
		vdiID := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vdi-import-vhd", 10*units.MB)

		// Create a test VHD disk image
		diskPath := createTestDiskImage(t, "vpc", 10*units.MB)
		defer os.Remove(diskPath)

		// Open the disk image file
		file, err := os.Open(diskPath)
		require.NoError(t, err, "opening test disk should succeed")
		defer file.Close()

		// Get file size
		fileInfo, err := file.Stat()
		require.NoError(t, err, "getting file info should succeed")

		// Import the disk into the VDI
		err = client.VDI().Import(ctx, vdiID, payloads.VDIFormatVHD, file, fileInfo.Size())
		require.NoError(t, err, "importing VHD disk should succeed")

		// Verify the VDI exists and has expected properties
		vdi, err := client.VDI().Get(ctx, vdiID)
		require.NoError(t, err, "getting imported VDI should succeed")
		assert.Equal(t, vdiID, vdi.ID, "VDI ID should match")
		assert.Equal(t, testPrefix+"vdi-import-vhd", vdi.NameLabel, "VDI name should match")
		assert.Greater(t, vdi.Size, int64(0), "VDI should have non-zero size")
	})
}
