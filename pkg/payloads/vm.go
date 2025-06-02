package payloads

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
)

// VMQueryOptions represents the query parameters for VM listing operations.
// This provides a type-safe way to build REST API query parameters.
type VMQueryOptions struct {
	Fields []string `json:"fields,omitempty"` // Fields to return in the response
	Filter string   `json:"filter,omitempty"` // Filter string in field:value format
	Limit  int      `json:"limit,omitempty"`  // Maximum number of results to return
}

// ToMap converts VMQueryOptions to a map[string]any for use with the REST API.
func (q *VMQueryOptions) ToMap() map[string]any {
	if q == nil {
		return map[string]any{}
	}

	options := make(map[string]any)

	if len(q.Fields) > 0 {
		options[VMQueryFields] = strings.Join(q.Fields, ",")
	}
	if q.Filter != "" {
		options[VMQueryFilter] = q.Filter
	}
	if q.Limit > 0 {
		options[VMQueryLimit] = q.Limit
	}

	return options
}

// VMFilter defines the available filtering options for VM queries.
// Use this struct with BuildFilterOptions() or construct the options map directly.
type VMFilter struct {
	PowerState string `json:"power_state,omitempty"` // Filter by VM power state (Running, Halted, Paused, Suspended)
	NameLabel  string `json:"name_label,omitempty"`  // Filter by VM name (supports partial matching)
	PoolID     string `json:"$poolId,omitempty"`     // Filter by pool ID
	Tags       string `json:"tags,omitempty"`        // Filter by tags
}

// REST API query parameter constants for VM endpoints
const (
	// Query parameter names
	VMQueryFields = "fields" // Specify which fields to return (comma-separated)
	VMQueryFilter = "filter" // Filter VMs using field:value format
	VMQueryLimit  = "limit"  // Limit number of results returned
)

// Common field names for the fields query parameter
const (
	VMFieldNameLabel  = "name_label"
	VMFieldPowerState = "power_state"
	VMFieldUUID       = "uuid"
	VMFieldPoolID     = "$poolId"
	VMFieldTags       = "tags"
	VMFieldMemory     = "memory"
	VMFieldCPUs       = "CPUs"
	VMFieldAddresses  = "addresses"
	VMFieldTemplate   = "template"
)

// Filter format examples:
// - Single filter: "power_state:Running"
// - Multiple filters: "power_state:Running,tags:production" (comma-separated)
// - Name matching: "name_label:web-server"
// - Pool filtering: "$poolId:pool-uuid"

// Power state constants for VM filtering and status
const (
	PowerStateHalted    = "Halted"
	PowerStateRunning   = "Running"
	PowerStatePaused    = "Paused"
	PowerStateSuspended = "Suspended"
)

/*
Videoram is represented as an integer, but sometimes comes as a string in the API response.
Therefore, we need to handle both formats by parsing it as a string when necessary and
converting it to an integer.
*/
type Videoram int

func (v *Videoram) UnmarshalJSON(data []byte) error {
	var intValue int
	if err := json.Unmarshal(data, &intValue); err == nil {
		*v = Videoram(intValue)
		return nil
	}

	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err != nil {
		return err
	}

	if stringValue == "" {
		*v = 0
		return nil
	}

	intValue, err := strconv.Atoi(stringValue)
	if err != nil {
		return err
	}

	*v = Videoram(intValue)
	return nil
}

type VM struct {
	// Core Identifiers & Type
	ID       uuid.UUID `json:"id"`
	UUID     string    `json:"uuid"`
	Type     string    `json:"type,omitempty"`
	Template uuid.UUID `json:"template,omitempty"`
	XapiRef  string    `json:"_xapiRef,omitempty"`

	NameLabel       string `json:"name_label"`
	NameDescription string `json:"name_description"`
	PowerState      string `json:"power_state,omitempty"`

	// Hardware Configuration
	Memory         Memory `json:"memory"`
	CPUs           CPUs   `json:"CPUs"`
	CoresPerSocket int    `json:"coresPerSocket,omitempty"`

	VIFs  []string `json:"VIFs,omitempty"`
	VBDs  []string `json:"$VBDs,omitempty"`
	VGPUs []string `json:"VGPUs,omitempty"`
	VTPMs []string `json:"VTPMs,omitempty"`

	Tags               []string `json:"tags,omitempty"`
	AutoPoweron        bool     `json:"auto_poweron"`
	HA                 string   `json:"high_availability,omitempty"`
	VirtualizationMode string   `json:"virtualizationMode,omitempty"`
	StartDelay         int      `json:"startDelay,omitempty"`
	ExpNestedHvm       bool     `json:"expNestedHvm,omitempty"`
	Boot               Boot     `json:"boot"`
	SecureBoot         bool     `json:"secureBoot,omitempty"`

	Videoram          Videoram          `json:"videoram,omitempty"`
	Vga               string            `json:"vga,omitempty"`
	XenstoreData      map[string]string `json:"xenStoreData,omitempty"`
	BlockedOperations map[string]string `json:"blockedOperations,omitempty"`

	// State & Metadata
	Addresses         map[string]string `json:"addresses,omitempty"`
	BiosStrings       map[string]string `json:"bios_strings,omitempty"`
	OsVersion         *OsVersion        `json:"os_version,omitempty"`
	InstallTime       int64             `json:"installTime,omitempty"`
	StartTime         *int64            `json:"startTime,omitempty"`
	CurrentOperations map[string]string `json:"current_operations,omitempty"`
	OtherConfig       map[string]string `json:"other,omitempty"`

	PoolID    uuid.UUID `json:"$poolId,omitempty"`
	Container string    `json:"$container,omitempty"`
}

type Memory struct {
	Dynamic []int64 `json:"dynamic,omitempty"`
	Static  []int64 `json:"static,omitempty"`
	Size    int64   `json:"size,omitempty"`
	Order   string  `json:"order,omitempty"`
}

type CPUs struct {
	Number int `json:"number"`
	Max    int `json:"max,omitempty"`
}

type Boot struct {
	Firmware string `json:"firmware,omitempty"`
	Order    string `json:"order,omitempty"`
}

type OsVersion struct {
	Name   string `json:"name,omitempty"`
	Uname  string `json:"uname,omitempty"`
	Distro string `json:"distro,omitempty"`
	Major  string `json:"major,omitempty"`
	Minor  string `json:"minor,omitempty"`
}
