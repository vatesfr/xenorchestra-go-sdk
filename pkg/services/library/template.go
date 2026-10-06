package library

import (
	"context"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen --build_flags=--mod=mod --destination mock/template.go . Template
type Template interface {
	// Get retrieves a VM template by its ID.
	// Parameters:
	//   - id: ID of the template to retrieve. A template id is the composite
	//     "<poolId>-<templateUuid>" string returned by GET /vm-templates, not
	//     a plain UUID, which is why this method takes a string.
	// Returns the template details or an error if the operation fails.
	Get(ctx context.Context, id string) (*payloads.Template, error)

	// GetAll retrieves VM templates with configurable limit and filtering.
	// Parameters:
	//   - limit: maximum number of templates to return (0 for no limit)
	//   - filter: filter string for template selection (empty for no filter)
	// Returns all matching templates or an error if the operation fails.
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Template, error)
}
