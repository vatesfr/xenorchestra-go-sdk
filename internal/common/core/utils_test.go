package core

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPathBuilder(t *testing.T) {
	t.Run("empty builder returns empty string", func(t *testing.T) {
		builder := NewPathBuilder()
		assert.Equal(t, "", builder.Build())
	})

	t.Run("single resource", func(t *testing.T) {
		builder := NewPathBuilder().Resource("vms")
		assert.Equal(t, "vms", builder.Build())
	})

	t.Run("resource with ID", func(t *testing.T) {
		id := uuid.Must(uuid.FromString("12345678-1234-1234-1234-123456789012"))
		builder := NewPathBuilder().Resource("vms").ID(id)
		assert.Equal(t, "vms/12345678-1234-1234-1234-123456789012", builder.Build())
	})

	t.Run("resource with string ID", func(t *testing.T) {
		builder := NewPathBuilder().Resource("tasks").IDString("abc123")
		assert.Equal(t, "tasks/abc123", builder.Build())
	})

	t.Run("resource with action", func(t *testing.T) {
		builder := NewPathBuilder().Resource("vms").IDString("123").Action("start")
		assert.Equal(t, "vms/123/start", builder.Build())
	})

	t.Run("resource with underscore placeholder", func(t *testing.T) {
		builder := NewPathBuilder().Resource("vms").IDString("_")
		assert.Equal(t, "vms/_", builder.Build())
	})

	t.Run("resource with actions group", func(t *testing.T) {
		builder := NewPathBuilder().Resource("vms").IDString("_").ActionsGroup()
		assert.Equal(t, "vms/_/actions", builder.Build())
	})

	t.Run("complex path", func(t *testing.T) {
		builder := NewPathBuilder().
			Resource("vms").
			IDString("_").
			ActionsGroup().
			Action("start")
		assert.Equal(t, "vms/_/actions/start", builder.Build())
	})

	t.Run("pool action path", func(t *testing.T) {
		id := uuid.Must(uuid.FromString("12345678-1234-1234-1234-123456789012"))
		builder := NewPathBuilder().
			Resource("pools").
			ID(id).
			ActionsGroup().
			Action("create_vm")
		assert.Equal(t, "pools/12345678-1234-1234-1234-123456789012/actions/create_vm", builder.Build())
	})
}

func TestFormatPath(t *testing.T) {
	id := uuid.Must(uuid.FromString("12345678-1234-1234-1234-123456789012"))
	path := FormatPath("vms", id)
	assert.Equal(t, "vms/12345678-1234-1234-1234-123456789012", path)
}

func TestFormatActionPath(t *testing.T) {
	path := FormatActionPath("vms", "start")
	assert.Equal(t, "vms/_/actions/start", path)
}
