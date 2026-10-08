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

// findTemplateForTest returns the template whose REST id matches
// intTests.testTemplateID, or the first template of the pool when it is not
// part of the collection.
func findTemplateForTest(t *testing.T, ctx context.Context, client library.Library) *payloads.Template {
	t.Helper()
	templates, err := client.Template().GetAll(ctx, 0, "")
	require.NoError(t, err)
	require.NotEmpty(t, templates, "expected at least one template in the pool")

	for _, tmpl := range templates {
		if tmpl.ID == intTests.testTemplateID {
			return tmpl
		}
	}
	return templates[0]
}

// findUUIDTemplateForTest returns a template whose REST id is a plain UUID,
// i.e. a non-default template. Sub-resource methods (tags, tasks) are
// addressed by the bare template UUID, which only equals the REST id for
// non-default templates (default templates use a composite
// "<poolUuid>-<templateUuid>" id), so tests relying on them need such a
// template and are skipped when none is available.
func findUUIDTemplateForTest(t *testing.T, ctx context.Context, client library.Library) *payloads.Template {
	t.Helper()
	templates, err := client.Template().GetAll(ctx, 0, "")
	require.NoError(t, err)
	require.NotEmpty(t, templates, "expected at least one template in the pool")

	for _, tmpl := range templates {
		if _, err := uuid.FromString(tmpl.ID); err == nil {
			return tmpl
		}
	}

	t.Skip("no non-default template found in the pool; template tags/tasks require a template whose id is a plain UUID")
	return nil
}

func TestTemplateGetAll(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	templates, err := client.Template().GetAll(ctx, 0, "")
	require.NoError(t, err)
	require.NotEmpty(t, templates, "GetAll should return at least one template")

	tmpl := templates[0]
	assert.Equal(t, payloads.ResourceTypeVMTemplate, tmpl.Type, "template type should be VM-template")
	assert.NotEmpty(t, tmpl.ID, "template ID should be set")
	assert.NotEqual(t, uuid.Nil, tmpl.UUID, "template UUID should be set")
	assert.NotEmpty(t, tmpl.NameLabel, "template name label should be set")
	assert.NotEqual(t, uuid.Nil, tmpl.Pool, "template pool should be set")

	t.Run("WithLimit", func(t *testing.T) {
		t.Parallel()
		templates, err := client.Template().GetAll(ctx, 1, "")
		require.NoError(t, err)
		assert.Len(t, templates, 1, "GetAll with limit=1 should return exactly one template")
	})

	t.Run("WithFilter", func(t *testing.T) {
		t.Parallel()
		filter := "uuid:" + tmpl.UUID.String()
		templates, err := client.Template().GetAll(ctx, 0, filter)
		require.NoError(t, err)
		require.Len(t, templates, 1, "GetAll filtered by uuid should return exactly one template")
		assert.Equal(t, tmpl.ID, templates[0].ID, "filtered template ID should match")
	})
}

func TestTemplateGet(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	tmpl := findTemplateForTest(t, ctx, client)

	t.Run("with valid ID", func(t *testing.T) {
		t.Parallel()
		got, err := client.Template().Get(ctx, tmpl.ID)
		require.NoErrorf(t, err, "failed to get template %s", tmpl.ID)
		assert.Equal(t, tmpl.ID, got.ID, "template ID should match")
		assert.Equal(t, tmpl.UUID, got.UUID, "template UUID should match")
		assert.Equal(t, tmpl.NameLabel, got.NameLabel, "template name label should match")
		assert.Equal(t, payloads.ResourceTypeVMTemplate, got.Type, "template type should be VM-template")
		assert.NotEmpty(t, got.NameLabel, "template name label should be populated")
	})

	t.Run("with invalid ID", func(t *testing.T) {
		t.Parallel()
		_, err := client.Template().Get(ctx, uuid.Must(uuid.NewV4()).String())
		require.Error(t, err, "expected error when fetching template with invalid ID")
	})
}

func templateTagExists(ctx context.Context, client library.Library, templateID, tag string) bool {
	tmpl, err := client.Template().Get(ctx, templateID)
	if err != nil {
		return false
	}
	return slices.Contains(tmpl.Tags, tag)
}

func TestTemplateTags(t *testing.T) {
	ctx, client, testPrefix := SetupTestContext(t)

	tmpl := findUUIDTemplateForTest(t, ctx, client)
	// For non-default templates the REST id is the bare template UUID.
	templateUUID, err := uuid.FromString(tmpl.ID)
	require.NoError(t, err)

	t.Run("AddTag", func(t *testing.T) {
		t.Parallel()
		tag := testPrefix + "tag"
		require.NoError(t, client.Template().AddTag(ctx, templateUUID, tag), "adding tag should succeed")
		require.Eventually(t, func() bool {
			return templateTagExists(ctx, client, tmpl.ID, tag)
		}, 1*time.Minute, 2*time.Second, "tag should be attached to the template")
	})

	t.Run("RemoveTag", func(t *testing.T) {
		t.Parallel()
		tag := testPrefix + "remove-tag"
		require.NoError(t, client.Template().AddTag(ctx, templateUUID, tag), "setup tag addition should succeed")
		require.Eventually(t, func() bool {
			return templateTagExists(ctx, client, tmpl.ID, tag)
		}, 1*time.Minute, 2*time.Second, "tag should be attached to the template")

		require.NoError(t, client.Template().RemoveTag(ctx, templateUUID, tag), "removing tag should succeed")
		require.Eventually(t, func() bool {
			return !templateTagExists(ctx, client, tmpl.ID, tag)
		}, 1*time.Minute, 2*time.Second, "tag should be removed from the template")
	})
}

func TestTemplateGetTasks(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	tmpl := findUUIDTemplateForTest(t, ctx, client)
	// For non-default templates the REST id is the bare template UUID.
	templateUUID, err := uuid.FromString(tmpl.ID)
	require.NoError(t, err)

	t.Run("GetTasks", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.Template().GetTasks(ctx, templateUUID, 0, "")
		require.NoError(t, err)
		require.NotNil(t, tasks)
	})

	t.Run("GetTasksWithLimit", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.Template().GetTasks(ctx, templateUUID, 1, "")
		require.NoError(t, err)
		require.NotNil(t, tasks)
		assert.LessOrEqual(t, len(tasks), 1, "GetTasks with limit=1 should return at most one task")
	})

	t.Run("GetTasksInvalidTemplate", func(t *testing.T) {
		t.Parallel()
		_, err := client.Template().GetTasks(ctx, uuid.Must(uuid.NewV4()), 0, "")
		require.Error(t, err, "expected error when fetching tasks of an unknown template")
	})
}
