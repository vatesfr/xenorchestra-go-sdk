package payloads

// ResourceType represents the type field common to all XO API resources.
type ResourceType string

// Resource type constants
const (
	ResourceTypeVBD  ResourceType = "VBD"
	ResourceTypeVDI  ResourceType = "VDI"
	ResourceTypePool ResourceType = "pool"
	ResourceTypeHost ResourceType = "host"
	ResourceTypeVM   ResourceType = "VM"
	ResourceTypePBD  ResourceType = "PBD"
	ResourceTypeSR   ResourceType = "SR"
)

var resourceTypePathMap = map[ResourceType]string{
	ResourceTypeVBD:  "vbds",
	ResourceTypeVDI:  "vdis",
	ResourceTypePool: "pools",
	ResourceTypeHost: "hosts",
	ResourceTypeVM:   "vms",
	ResourceTypePBD:  "pbds",
	ResourceTypeSR:   "srs",
}

// Path returns the API path segment corresponding to the resource type.
func (rt ResourceType) Path() string {
	return resourceTypePathMap[rt]
}
