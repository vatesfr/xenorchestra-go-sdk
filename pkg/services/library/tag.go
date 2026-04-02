package library

import (
	"context"

	"github.com/gofrs/uuid"
)

// Taggable is implemented by all resources that support tag operations.
type Taggable interface {
	// AddTag adds a tag to the resource.
	// Parameters:
	//   - id: ID of the resource to which the tag will be added
	//   - tag: tag string to add
	// Returns an error if the operation fails.
	AddTag(ctx context.Context, id uuid.UUID, tag string) error

	// RemoveTag removes a tag from the resource.
	// Parameters:
	//   - id: ID of the resource from which the tag will be removed
	//   - tag: tag string to remove
	// Returns an error if the operation fails.
	RemoveTag(ctx context.Context, id uuid.UUID, tag string) error
}
