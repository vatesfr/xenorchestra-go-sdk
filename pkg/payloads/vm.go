/*
TODO: REMOVE THIS COMMENT. Payloads package is all the contracts needed for
to interact with Xen Orchestra Rest API. We won't keep the payload model,
inside with the interface and implementation, it will be easier to maintain
*/
package payloads

import (
	"encoding/json"
	"strconv"

	"github.com/gofrs/uuid"
)

/*
As videoram is an int but comes as a string from the response sometimes,
we need to parse it as a string first and then convert it to an int.
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

// Introducing stronger type for UUID by using a package rather than a string.
type VM struct {
	ID                 uuid.UUID         `json:"id,omitempty"`
	Template           uuid.UUID         `json:"template,omitempty"`
	NameLabel          string            `json:"name_label"`
	NameDescription    string            `json:"name_description"`
	PowerState         string            `json:"power_state,omitempty"`
	Memory             Memory            `json:"memory"`
	CPUs               CPUs              `json:"CPUs"`
	VIFs               []string          `json:"VIFs,omitempty"`
	VBDs               []string          `json:"$VBDs,omitempty"`
	Tags               []string          `json:"tags,omitempty"`
	AutoPoweron        bool              `json:"auto_poweron"`
	HA                 string            `json:"high_availability,omitempty"`
	VirtualizationMode string            `json:"virtualizationMode,omitempty"`
	StartDelay         int               `json:"startDelay,omitempty"`
	ExpNestedHvm       bool              `json:"expNestedHvm,omitempty"`
	Boot               Boot              `json:"boot"`
	Videoram           Videoram          `json:"videoram,omitempty"`
	Vga                string            `json:"vga,omitempty"`
	XenstoreData       map[string]string `json:"xenStoreData,omitempty"`
	BlockedOperations  map[string]string `json:"blockedOperations,omitempty"`
	PoolID             uuid.UUID         `json:"$poolId,omitempty"`
	Container          string            `json:"$container,omitempty"`
}

type Memory struct {
	Dynamic []int64 `json:"dynamic,omitempty"`
	Static  []int64 `json:"static,omitempty"`
	Size    int64   `json:"size,omitempty"`
}

type CPUs struct {
	Number int `json:"number"`
	Max    int `json:"max,omitempty"`
}

type Boot struct {
	Firmware string `json:"firmware,omitempty"`
	Order    string `json:"order,omitempty"`
}

type VMFilter struct {
	PowerState string `json:"power_state,omitempty"`
	NameLabel  string `json:"name_label,omitempty"`
	PoolID     string `json:"$poolId,omitempty"`
	Tags       string `json:"tags,omitempty"`
}

const (
	PowerStateHalted    = "Halted"
	PowerStateRunning   = "Running"
	PowerStatePaused    = "Paused"
	PowerStateSuspended = "Suspended"
)
