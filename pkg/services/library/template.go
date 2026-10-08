package library

import (
	"context"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen --build_flags=--mod=mod --destination mock/template.go . Template
type Template interface {
	// Get retrieves a VM template by its ID.
	// Parameters:
	//   - id: ID of the template to retrieve. A template id is a string: for
	//     default templates it is the composite "<poolUuid>-<templateUuid>"
	//     value, for non-default templates it is the bare template UUID. It
	//     is the "id" value returned by GET /vm-templates.
	// Returns the template details or an error if the operation fails.
	Get(ctx context.Context, id string) (*payloads.Template, error)

	// GetAll retrieves VM templates with configurable limit and filtering.
	// Parameters:
	//   - limit: maximum number of templates to return (0 for no limit)
	//   - filter: filter string for template selection (empty for no filter)
	// Returns all matching templates or an error if the operation fails.
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Template, error)

	// Taggable and Taskable are addressed by the bare template UUID
	// (Template.UUID), not by the composite REST id used for default
	// templates.
	Taggable

	Taskable
}
