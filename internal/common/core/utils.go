package core

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
)

// PathBuilder helps construct API endpoint paths in a consistent way.
// It provides a fluent interface for building paths like "vms/123/start"
// or "vms/_/actions/snapshot". I find the "vms"+id.String() a little bit
// ugly, so I created this helper to be used in the different services.
type PathBuilder struct {
	segments []string
}

func NewPathBuilder() *PathBuilder {
	return &PathBuilder{segments: []string{}}
}

// Resource adds a resource type to the path (e.g., "vms", "tasks").
func (p *PathBuilder) Resource(resource string) *PathBuilder {
	p.segments = append(p.segments, resource)
	return p
}

// ID adds a UUID resource ID to the path.
func (p *PathBuilder) ID(id uuid.UUID) *PathBuilder {
	p.segments = append(p.segments, id.String())
	return p
}

// IDString adds a string ID to the path when you don't have a UUID.
// It could be the UUID v4 from the package we are using however it could
// be any string ID for other cases...
func (p *PathBuilder) IDString(id string) *PathBuilder {
	p.segments = append(p.segments, id)
	return p
}

// Action adds an action to the path (e.g., "start", "suspend").
func (p *PathBuilder) Action(action string) *PathBuilder {
	p.segments = append(p.segments, action)
	return p
}

// Wildcard adds a wildcard segment ("_") to the path.
// This is used in XO API for actions that apply to any resource of a type.
func (p *PathBuilder) Wildcard() *PathBuilder {
	p.segments = append(p.segments, "_")
	return p
}

// ActionsGroup adds an "actions" segment to the path.
// This is used in XO API to group actions on a resource.
func (p *PathBuilder) ActionsGroup() *PathBuilder {
	p.segments = append(p.segments, "actions")
	return p
}

// Build returns the constructed path with segments joined by "/".
func (p *PathBuilder) Build() string {
	return strings.Join(p.segments, "/")
}

// FormatPath is a convenience function for simple resource/ID paths.
// It creates paths like "vms/12345678-1234-1234-1234-123456789012".
func FormatPath(resource string, id uuid.UUID) string {
	return fmt.Sprintf("%s/%s", resource, id.String())
}

// FormatActionPath is a convenience function for resource wildcard action paths.
// It creates paths like "vms/_/actions/start".
func FormatActionPath(resource string, action string) string {
	return fmt.Sprintf("%s/_/actions/%s", resource, action)
}
