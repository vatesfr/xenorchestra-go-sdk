package integration

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestPBDGet(t *testing.T) {
	t.Parallel()
	ctx, client, _ := SetupTestContext(t)

	// Retrieve all PBDs to get a known valid ID to test Get with.
	pbds, err := client.PBD().GetAll(ctx, 1, "")
	require.NoError(t, err, "GetAll should succeed to seed a valid PBD ID")
	require.NotEmpty(t, pbds, "there should be at least one PBD in the infrastructure")

	pbdID := pbds[0].UUID
	require.NotEqual(t, uuid.Nil, pbdID, "seed PBD ID should be valid")

	t.Run("GetByValidID", func(t *testing.T) {
		t.Parallel()
		pbd, err := client.PBD().Get(ctx, pbdID)
		require.NoError(t, err, "fetching PBD by valid ID should succeed")
		require.NotNil(t, pbd)
		assert.Equal(t, pbdID, pbd.UUID, "PBD UUID should match requested ID")
		assert.NotEqual(t, uuid.Nil, pbd.Host, "PBD should reference a host")
		assert.NotEqual(t, uuid.Nil, pbd.SR, "PBD should reference an SR")
	})

	t.Run("GetByInvalidID", func(t *testing.T) {
		t.Parallel()
		_, err := client.PBD().Get(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
		require.Error(t, err, "expected error when fetching PBD with non-existent ID")
	})
}

func TestPBDGetAll(t *testing.T) {
	t.Parallel()
	ctx, client, _ := SetupTestContext(t)

	t.Run("NoLimit", func(t *testing.T) {
		t.Parallel()
		pbds, err := client.PBD().GetAll(ctx, 0, "")
		require.NoError(t, err)
		require.NotNil(t, pbds)
		assert.NotEmpty(t, pbds, "GetAll should return at least one PBD")
	})

	t.Run("WithLimit", func(t *testing.T) {
		t.Parallel()
		pbds, err := client.PBD().GetAll(ctx, 2, "")
		require.NoError(t, err)
		require.NotNil(t, pbds)
		assert.Len(t, pbds, 2, "GetAll with limit=2 should return exactly two PBDs")
	})

	t.Run("WithAttachedFilter", func(t *testing.T) {
		t.Parallel()
		pbds, err := client.PBD().GetAll(ctx, 0, "attached?")
		require.NoError(t, err)
		require.NotNil(t, pbds)
		for _, pbd := range pbds {
			assert.True(t, pbd.Attached, "all returned PBDs should be attached when filtering by attached?")
		}
	})

	t.Run("WithFilterNoResult", func(t *testing.T) {
		t.Parallel()
		// Filter by a non-existent host UUID to get zero results.
		pbds, err := client.PBD().GetAll(ctx, 0, "host:123e4567-e89b-12d3-a456-426655440000")
		require.NoError(t, err)
		require.NotNil(t, pbds)
		assert.Len(t, pbds, 0, "filter by non-existent host ID should return no PBDs")
	})
}

// TestPBDPlugUnplug tests the Plug and Unplug actions on a real PBD.
// Set XOA_TEST_PBD_ID to the UUID of a PBD that is safe to temporarily disconnect
// (e.g. removable storage on a test host). The PBD is restored to its original
// Attached state after the test via t.Cleanup.
func TestPBDPlugUnplug(t *testing.T) {
	t.Parallel()
	ctx, client, _ := SetupTestContext(t)

	// This PBD must be safe to temporarily disconnect (e.g. removable storage on a test host).
	if intTests.testPBD == uuid.Nil {
		t.Skip("XOA_TEST_PBD_ID is not set: skipping PBD plug/unplug tests")
	}

	// Read the initial state so we can restore it in Cleanup.
	initial, err := client.PBD().Get(ctx, intTests.testPBD)
	require.NoErrorf(t, err, "failed to fetch PBD %s before plug/unplug tests", intTests.testPBD)

	// Restore the PBD to its original Attached state after the test.
	t.Cleanup(func() {
		// Use the long-lived parent context so cleanup is not cut short by the test timeout.
		pbd, err := client.PBD().Get(intTests.ctx, intTests.testPBD)
		if err != nil {
			t.Logf("cleanup: failed to fetch PBD %s state: %v", intTests.testPBD, err)
			return
		}
		switch {
		case initial.Attached && !pbd.Attached:
			taskID, err := client.PBD().Plug(intTests.ctx, intTests.testPBD)
			if err != nil {
				t.Logf("cleanup: failed to plug PBD %s: %v", intTests.testPBD, err)
				return
			}
			waitForTask(t, intTests.ctx, client, taskID)
		case !initial.Attached && pbd.Attached:
			taskID, err := client.PBD().Unplug(intTests.ctx, intTests.testPBD)
			if err != nil {
				t.Logf("cleanup: failed to unplug PBD %s: %v", intTests.testPBD, err)
				return
			}
			waitForTask(t, intTests.ctx, client, taskID)
		}
	})

	// The Unplug sub-test requires the PBD to be attached first.
	if !initial.Attached {
		setupTaskID, err := client.PBD().Plug(ctx, intTests.testPBD)
		require.NoErrorf(t, err, "setup: failed to plug PBD %s before Unplug sub-test", intTests.testPBD)
		task := waitForTask(t, ctx, client, setupTaskID)
		require.Equalf(t, payloads.Success, task.Status,
			"setup: plug task should succeed before Unplug sub-test: %v", task.Result.Message)
	}

	// Sub-tests are intentionally sequential: Unplug must run before Plug so that
	// each sub-test starts from a known state.
	t.Run("Unplug", func(t *testing.T) {
		taskID, err := client.PBD().Unplug(ctx, intTests.testPBD)
		require.NoError(t, err, "Unplug should not return an error")
		task := waitForTask(t, ctx, client, taskID)
		require.Equalf(t, payloads.Success, task.Status,
			"Unplug task should succeed: %v", task.Result.Message)

		require.Eventually(t, func() bool {
			pbd, err := client.PBD().Get(ctx, intTests.testPBD)
			return err == nil && !pbd.Attached
		}, 2*time.Minute, 5*time.Second, "PBD should be detached after Unplug")
	})

	t.Run("Plug", func(t *testing.T) {
		taskID, err := client.PBD().Plug(ctx, intTests.testPBD)
		require.NoError(t, err, "Plug should not return an error")
		task := waitForTask(t, ctx, client, taskID)
		require.Equalf(t, payloads.Success, task.Status,
			"Plug task should succeed: %v", task.Result.Message)

		require.Eventually(t, func() bool {
			pbd, err := client.PBD().Get(ctx, intTests.testPBD)
			return err == nil && pbd.Attached
		}, 2*time.Minute, 5*time.Second, "PBD should be attached after Plug")
	})
}
