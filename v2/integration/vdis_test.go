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
