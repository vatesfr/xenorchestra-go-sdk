package payloads

import "github.com/gofrs/uuid"

// Schedule represents a timing configuration that defines when backup jobs should be executed.
// Schedules use cron expressions to define recurring execution times and can be associated
// with multiple backup jobs. They support timezone-specific scheduling and can be enabled/disabled.
type Schedule struct {
	ID       uuid.UUID `json:"id"`             // Unique identifier for the schedule
	JobID    uuid.UUID `json:"jobId"`          // Reference to the backup job that uses this schedule
	Name     string    `json:"name,omitempty"` // Human-readable name for the schedule
	Cron     string    `json:"cron"`           // Cron expression defining when the job should run (e.g., "0 2 * * *")
	Enabled  bool      `json:"enabled"`        // Whether this schedule is currently active
	Timezone string    `json:"timezone"`       // Timezone for interpreting the cron expression (e.g., "America/New_York")
}
