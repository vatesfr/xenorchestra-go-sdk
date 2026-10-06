package payloads

import "github.com/gofrs/uuid"

// Template represents a VM template as exposed by the REST resource
// "vm-templates". Unlike VMs (and most other resources), a template's REST id
// is not a plain UUID but the composite "<poolId>-<templateUuid>" value
// (73 characters), so the service that wraps this resource takes string ids.
// The bare template UUID (the one create_vm expects) is available in UUID
// when the server returns it.
type Template struct {
	// ID is the template's REST id, e.g.
	// "aaaaaaaa-bbbb-cccc-dddd-000000000001-6959dfe8-534c-4c58-8a8c-3c3792293543".
	ID string `json:"id,omitempty"`
	// UUID is the bare template UUID, when the server returns it. It is the
	// value create_vm (the "template" parameter) expects.
	UUID uuid.UUID `json:"uuid,omitempty"`
	// Type is the resource type, "VM-template" for a template.
	Type ResourceType `json:"type,omitempty"`

	NameLabel           string `json:"name_label,omitempty"`
	NameDescription     string `json:"name_description,omitempty"`
	PowerState          string `json:"power_state,omitempty"`
	IsDefault           bool   `json:"isDefaultTemplate,omitempty"`
	IsFirmwareSupported bool   `json:"isFirmwareSupported,omitempty"`

	Memory            Memory            `json:"memory,omitempty"`
	CPUs              CPUs              `json:"CPUs,omitempty"`
	Boot              Boot              `json:"boot,omitempty"`
	VIFs              []string          `json:"VIFs,omitempty"`
	Tags              []string          `json:"tags,omitempty"`
	AutoPoweron       bool              `json:"auto_poweron,omitempty"`
	HighAvailability  string            `json:"high_availability,omitempty"`
	CurrentOperations map[string]string `json:"current_operations,omitempty"`
	MainIpAddress     string            `json:"mainIpAddress,omitempty"`
	TemplateInfo      TemplateInfo      `json:"template_info,omitempty"`
	Creation          Creation          `json:"creation,omitempty"`

	// Pool and PoolID are the pool the template belongs to. The server
	// returns both "$pool" and "$poolId" (same value); both are exposed.
	Pool   uuid.UUID `json:"$pool,omitempty"`
	PoolID uuid.UUID `json:"$poolId,omitempty"`
	// Container is the container the template is bound to (the pool, for a
	// template), as a "$container" reference.
	Container uuid.UUID `json:"$container,omitempty"`
}

// TemplateInfo is the provisioning/install information attached to a template
// (the "template_info" field).
type TemplateInfo struct {
	Arch              string         `json:"arch,omitempty"`
	Disks             []TemplateDisk `json:"disks,omitempty"`
	InstallMethods    []string       `json:"install_methods,omitempty"`
	InstallRepository string         `json:"install_repository,omitempty"`
}

// TemplateDisk describes one disk of a template to be provisioned.
type TemplateDisk struct {
	Bootable bool   `json:"bootable,omitempty"`
	Size     int64  `json:"size,omitempty"`
	SR       string `json:"SR,omitempty"`
	Type     string `json:"type,omitempty"`
}
