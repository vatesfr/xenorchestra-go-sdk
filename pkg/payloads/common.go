package payloads

import "github.com/gofrs/uuid"

type CreateResponse struct {
	ID uuid.UUID `json:"id"`
}
