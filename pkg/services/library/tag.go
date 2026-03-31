package library

import (
	"context"

	"github.com/gofrs/uuid"
)

type Taggable interface {
	Tag() TagService
}

type TagService interface {

	// Add adds a tag to the SR.
	// Parameters:
	//   - id: ID of the SR
	//   - tag: tag string to add
	// Returns an error if the operation fails.
	Add(ctx context.Context, id uuid.UUID, tag string) error

	// Remove removes a tag from the SR.
	// Parameters:
	//   - id: ID of the SR
	//   - tag: tag string to remove
	// Returns an error if the operation fails.
	Remove(ctx context.Context, id uuid.UUID, tag string) error
}
