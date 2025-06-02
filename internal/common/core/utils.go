package core

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
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

func ExtractTaskID(response string) payloads.TaskID {
	return payloads.TaskID(strings.TrimPrefix(response, "/rest/v0/tasks/"))
}

// CleanDuplicateV0Path removes the redundant "/rest/v0" from paths.
// This is needed because VM creation returns a path with "/rest/v0" prefix,
// but our client already includes "/v0/rest" in the base URL.
func CleanDuplicateV0Path(path string) string {
	if !strings.HasPrefix(path, "/") {
		return path
	}
	return strings.TrimPrefix(path, "/rest/v0/tasks/")
}

// VM Query Utility Functions

// NewVMQuery creates a new VMQueryOptions with optional initial values.
func NewVMQuery() *payloads.VMQueryOptions {
	return &payloads.VMQueryOptions{}
}

// WithFields adds field selection to the query.
func WithFields(q *payloads.VMQueryOptions, fields ...string) *payloads.VMQueryOptions {
	q.Fields = fields
	return q
}

// WithFilter adds a filter string to the query.
func WithFilter(q *payloads.VMQueryOptions, filter string) *payloads.VMQueryOptions {
	q.Filter = filter
	return q
}

// WithLimit adds a result limit to the query.
func WithLimit(q *payloads.VMQueryOptions, limit int) *payloads.VMQueryOptions {
	q.Limit = limit
	return q
}

// BuildFilter creates a filter string from individual field:value pairs.
func BuildFilter(filters ...string) string {
	return strings.Join(filters, ",")
}

// FilterByPowerState creates a power state filter string.
func FilterByPowerState(state string) string {
	return fmt.Sprintf("%s:%s", payloads.VMFieldPowerState, state)
}

// FilterByNameLabel creates a name label filter string.
func FilterByNameLabel(nameLabel string) string {
	return fmt.Sprintf("%s:%s", payloads.VMFieldNameLabel, nameLabel)
}

// FilterByPoolID creates a pool ID filter string.
func FilterByPoolID(poolID string) string {
	return fmt.Sprintf("%s:%s", payloads.VMFieldPoolID, poolID)
}

// FilterByTags creates a tags filter string.
func FilterByTags(tags string) string {
	return fmt.Sprintf("%s:%s", payloads.VMFieldTags, tags)
}

// BuildFilterFromStruct creates a filter string from a VMFilter struct.
func BuildFilterFromStruct(filter *payloads.VMFilter) string {
	if filter == nil {
		return ""
	}

	var filters []string

	if filter.PowerState != "" {
		filters = append(filters, FilterByPowerState(filter.PowerState))
	}
	if filter.NameLabel != "" {
		filters = append(filters, FilterByNameLabel(filter.NameLabel))
	}
	if filter.PoolID != "" {
		filters = append(filters, FilterByPoolID(filter.PoolID))
	}
	if filter.Tags != "" {
		filters = append(filters, FilterByTags(filter.Tags))
	}

	return BuildFilter(filters...)
}

// Quick builder functions for common queries

// QueryRunningVMs creates a query for running VMs with basic fields.
func QueryRunningVMs() *payloads.VMQueryOptions {
	query := NewVMQuery()
	WithFields(query, payloads.VMFieldNameLabel, payloads.VMFieldPowerState, payloads.VMFieldUUID)
	WithFilter(query, FilterByPowerState(payloads.PowerStateRunning))
	return query
}

// QueryVMsByPool creates a query for VMs in a specific pool.
func QueryVMsByPool(poolID string) *payloads.VMQueryOptions {
	query := NewVMQuery()
	WithFilter(query, FilterByPoolID(poolID))
	return query
}

// QueryVMsWithLimit creates a query with a result limit.
func QueryVMsWithLimit(limit int) *payloads.VMQueryOptions {
	query := NewVMQuery()
	WithLimit(query, limit)
	return query
}
