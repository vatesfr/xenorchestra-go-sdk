package payloads

import "github.com/gofrs/uuid"

type Schedule struct {
	ID       uuid.UUID `json:"id"`
	JobID    uuid.UUID `json:"jobId"`
	Name     string    `json:"name,omitempty"`
	Cron     string    `json:"cron"`
	Enabled  bool      `json:"enabled"`
	Timezone string    `json:"timezone"`
}
