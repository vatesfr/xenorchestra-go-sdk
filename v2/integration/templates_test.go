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

	t.Run("with valid ID", func(t *testing.T) {
		t.Parallel()
		got, err := client.Template().Get(ctx, intTests.testTemplate.ID)
		require.NoErrorf(t, err, "failed to get template %s", intTests.testTemplate.ID)
		assert.Equal(t, intTests.testTemplate.ID, got.ID, "template ID should match")
		assert.Equal(t, intTests.testTemplate.UUID, got.UUID, "template UUID should match")
		assert.Equal(t, intTests.testTemplate.NameLabel, got.NameLabel, "template name label should match")
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

	t.Run("AddTag", func(t *testing.T) {
		t.Parallel()
		tag := testPrefix + "tag"
		require.NoError(t, client.Template().AddTag(ctx, intTests.testTemplate.ID, tag), "adding tag should succeed")
		require.Eventually(t, func() bool {
			return templateTagExists(ctx, client, intTests.testTemplate.ID, tag)
		}, 1*time.Minute, 2*time.Second, "tag should be attached to the template")
	})

	t.Run("RemoveTag", func(t *testing.T) {
		t.Parallel()
		tag := testPrefix + "remove-tag"
		require.NoError(t, client.Template().AddTag(ctx, intTests.testTemplate.ID, tag), "setup tag addition should succeed")
		require.Eventually(t, func() bool {
			return templateTagExists(ctx, client, intTests.testTemplate.ID, tag)
		}, 1*time.Minute, 2*time.Second, "tag should be attached to the template")

		require.NoError(t, client.Template().RemoveTag(ctx, intTests.testTemplate.ID, tag), "removing tag should succeed")
		require.Eventually(t, func() bool {
			return !templateTagExists(ctx, client, intTests.testTemplate.ID, tag)
		}, 1*time.Minute, 2*time.Second, "tag should be removed from the template")
	})
}

func TestTemplateGetTasks(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	t.Run("GetTasks", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.Template().GetTasks(ctx, intTests.testTemplate.ID, 0, "")
		require.NoError(t, err)
		require.NotNil(t, tasks)
	})

	t.Run("GetTasksWithLimit", func(t *testing.T) {
		t.Parallel()
		tasks, err := client.Template().GetTasks(ctx, intTests.testTemplate.ID, 1, "")
		require.NoError(t, err)
		require.NotNil(t, tasks)
		assert.LessOrEqual(t, len(tasks), 1, "GetTasks with limit=1 should return at most one task")
	})

	t.Run("GetTasksInvalidTemplate", func(t *testing.T) {
		t.Parallel()
		_, err := client.Template().GetTasks(ctx, uuid.Must(uuid.NewV4()).String(), 0, "")
		require.Error(t, err, "expected error when fetching tasks of an unknown template")
	})
}
