package integration

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
)

func TestSRGet(t *testing.T) {
	t.Parallel()
	ctx, client, _ := SetupTestContext(t)

	t.Run("GetByValidID", func(t *testing.T) {
		t.Parallel()
		sr, err := client.SR().Get(ctx, intTests.testSR.ID)
		require.NoError(t, err, "fetching SR by valid ID should succeed")
		require.NotNil(t, sr)
		assert.Equal(t, intTests.testSR.ID, sr.UUID, "SR UUID should match requested ID")
		assert.NotEmpty(t, sr.NameLabel, "SR should have a name_label")
		assert.NotEmpty(t, sr.SRType, "SR should have a SR_type")
	})

	t.Run("GetByInvalidID", func(t *testing.T) {
		t.Parallel()
		_, err := client.SR().Get(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
		require.Error(t, err, "expected error when fetching SR with non-existent ID")
	})
}

func TestSRGetAll(t *testing.T) {
	t.Parallel()
	ctx, client, _ := SetupTestContext(t)

	t.Run("NoLimit", func(t *testing.T) {
		t.Parallel()
		srs, err := client.SR().GetAll(ctx, 0, "")
		require.NoError(t, err)
		require.NotNil(t, srs)
		assert.NotEmpty(t, srs, "GetAll should return at least one SR")
	})

	t.Run("WithLimit", func(t *testing.T) {
		t.Parallel()
		srs, err := client.SR().GetAll(ctx, 2, "")
		require.NoError(t, err)
		require.NotNil(t, srs)
		assert.LessOrEqual(t, len(srs), 2, "GetAll with limit=2 should return at most two SRs")
	})

	t.Run("WithSharedFilter", func(t *testing.T) {
		t.Parallel()
		srs, err := client.SR().GetAll(ctx, 0, "shared?")
		require.NoError(t, err)
		require.NotNil(t, srs)
		for _, sr := range srs {
			assert.True(t, sr.Shared, "all returned SRs should be shared when filtering by shared?")
		}
	})

	t.Run("WithFilterNoResult", func(t *testing.T) {
		t.Parallel()
		// Filter by a non-existent name_label to get zero results.
		srs, err := client.SR().GetAll(ctx, 0, "name_label:nonexistent-sr-zzzz-not-found")
		require.NoError(t, err)
		require.NotNil(t, srs)
		assert.Len(t, srs, 0, "filter by non-existent name_label should return no SRs")
	})
}

func TestSRGetTasks(t *testing.T) {
	t.Parallel()
	ctx, client, _ := SetupTestContext(t)

	t.Run("GetTasksForValidSR", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.SR().GetTasks(ctx, intTests.testSR.ID, 0, "")
		require.NoError(t, err, "fetching tasks for a valid SR ID should succeed")
		require.NotNil(t, tasks)
		// Tasks may be empty if none exist for this SR — that is a valid state.
	})

	t.Run("GetTasksWithLimit", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.SR().GetTasks(ctx, intTests.testSR.ID, 5, "")
		require.NoError(t, err, "fetching tasks with limit should succeed")
		require.NotNil(t, tasks)
		assert.LessOrEqual(t, len(tasks), 5, "GetTasks with limit=5 should return at most 5 tasks")
	})

	t.Run("GetTasksForInvalidSR", func(t *testing.T) {
		t.Parallel()
		_, err := client.SR().GetTasks(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"), 0, "")
		require.Error(t, err, "expected error when fetching tasks for non-existent SR ID")
	})
}

func TestSRScan(t *testing.T) {
	t.Parallel()
	ctx, client, _ := SetupTestContext(t)

	t.Run("ScanValidSR", func(t *testing.T) {
		t.Parallel()
		taskID, err := client.SR().Scan(ctx, intTests.testSR.ID)
		require.NoError(t, err, "Scan should not return an error")
		require.NotEmpty(t, taskID, "Scan should return a non-empty task ID")

		task := waitForTask(t, ctx, client, taskID)
		require.Equalf(t, payloads.Success, task.Status,
			"Scan task should succeed: %v", task.Result.Message)
	})

	t.Run("ScanInvalidSR", func(t *testing.T) {
		t.Parallel()
		// The POST succeeds (200) and returns a task ID even for a non-existent SR.
		// The task then fails asynchronously with result.message="no such object {id}".
		taskID, err := client.SR().Scan(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
		require.NoError(t, err, "Scan should return a task ID even for a non-existent SR")
		require.NotEmpty(t, taskID, "Scan should return a non-empty task ID")

		task := waitForTask(t, ctx, client, taskID)
		assert.Equal(t, payloads.Failure, task.Status, "Scan task for a non-existent SR should fail")
	})
}

func TestSRReclaimSpace(t *testing.T) {
	t.Parallel()
	ctx, client, _ := SetupTestContext(t)

	t.Run("ReclaimSpaceValidSR", func(t *testing.T) {
		t.Parallel()
		taskID, err := client.SR().ReclaimSpace(ctx, intTests.testSR.ID)
		require.NoError(t, err, "ReclaimSpace should not return an error")
		require.NotEmpty(t, taskID, "ReclaimSpace should return a non-empty task ID")

		task := waitForTask(t, ctx, client, taskID)
		require.Equalf(t, payloads.Success, task.Status,
			"ReclaimSpace task should succeed: %v", task.Result.Message)
	})

	t.Run("ReclaimSpaceInvalidSR", func(t *testing.T) {
		t.Parallel()
		// The POST succeeds (200) and returns a task ID even for a non-existent SR.
		// The task then fails asynchronously with result.message="no such object {id}".
		taskID, err := client.SR().ReclaimSpace(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
		require.NoError(t, err, "ReclaimSpace should return a task ID even for a non-existent SR")
		require.NotEmpty(t, taskID, "ReclaimSpace should return a non-empty task ID")

		task := waitForTask(t, ctx, client, taskID)
		assert.Equal(t, payloads.Failure, task.Status, "ReclaimSpace task for a non-existent SR should fail")
	})
}

func srTagExists(ctx context.Context, client library.Library, srID uuid.UUID, tag string) bool {
	sr, err := client.SR().Get(ctx, srID)
	if err != nil {
		return false
	}
	return slices.Contains(sr.Tags, tag)
}

func TestSRAddTag(t *testing.T) {
	ctx, client, prefix := SetupTestContext(t)

	tag := prefix + "tag"

	t.Cleanup(func() {
		_ = client.SR().Tag().Remove(ctx, intTests.testSR.ID, tag)
	})

	require.NoError(t, client.SR().Tag().Add(ctx, intTests.testSR.ID, tag), "adding tag should succeed")

	require.Eventually(t, func() bool {
		return srTagExists(ctx, client, intTests.testSR.ID, tag)
	}, 1*time.Minute, 2*time.Second, "tag should be attached to the SR")

	refreshed, err := client.SR().Get(ctx, intTests.testSR.ID)
	require.NoError(t, err)
	assert.Contains(t, refreshed.Tags, tag, "SR tags should contain the newly added tag")
}

func TestSRRemoveTag(t *testing.T) {
	ctx, client, prefix := SetupTestContext(t)

	tag := prefix + "remove-tag"

	require.NoError(t, client.SR().Tag().Add(ctx, intTests.testSR.ID, tag), "setup tag addition should succeed")

	require.Eventually(t, func() bool {
		return srTagExists(ctx, client, intTests.testSR.ID, tag)
	}, 1*time.Minute, 2*time.Second, "tag should be attached to the SR before removal")

	require.NoError(t, client.SR().Tag().Remove(ctx, intTests.testSR.ID, tag), "removing tag should succeed")

	require.Eventually(t, func() bool {
		return !srTagExists(ctx, client, intTests.testSR.ID, tag)
	}, 1*time.Minute, 2*time.Second, "tag should be removed from the SR")

	refreshed, err := client.SR().Get(ctx, intTests.testSR.ID)
	require.NoError(t, err)
	assert.NotContains(t, refreshed.Tags, tag, "SR tags should not contain the removed tag")
}
