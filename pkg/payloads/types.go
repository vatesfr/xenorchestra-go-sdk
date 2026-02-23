package payloads

// ResourceType represents the type field common to all XO API resources.
type ResourceType string

// Resource type constants
const (
	ResourceTypeVDI  ResourceType = "VDI"
	ResourceTypePool ResourceType = "pool"
	ResourceTypeHost ResourceType = "host"
	ResourceTypeVM   ResourceType = "VM"
)
