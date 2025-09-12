package payloads

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

type Status string

const (
	Success     Status = "success"
	Failure     Status = "failure"
	Interrupted Status = "interrupted"
	Pending     Status = "pending"
)

type APITime time.Time

func (t *APITime) UnmarshalJSON(data []byte) error {
	var timeStr string
	if err := json.Unmarshal(data, &timeStr); err == nil {
		parsedTime, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return fmt.Errorf("failed to parse time string: %v", err)
		}
		*t = APITime(parsedTime)
		return nil
	}

	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return fmt.Errorf("value is neither a valid time string nor a Unix timestamp: %v", err)
	}

	*t = APITime(time.UnixMilli(timestamp))
	return nil
}

func (t APITime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.RFC3339))
}

func (t APITime) Time() time.Time {
	return time.Time(t)
}

func (t APITime) String() string {
	return time.Time(t).String()
}

type DataMessage struct {
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

type Properties struct {
	UserID   string         `json:"userId,omitempty"`
	Type     string         `json:"type,omitempty"`
	Params   map[string]any `json:"params,omitempty"`
	ObjectID string         `json:"objectId,omitempty"`
	Name     string         `json:"name"`
	Method   string         `json:"method,omitempty"`
}

type Result struct {
	Code   any            `json:"code"` // Can be either a number or a string
	Data   map[string]any `json:"data,omitempty"`
	Params []any          `json:"params"`
	Call   struct {
		Method   string `json:"method"`
		Duration int64  `json:"duration"`
		Params   []any  `json:"params"`
	} `json:"call"`
	Message string    `json:"message,omitempty"`
	Name    string    `json:"name,omitempty"`
	Stack   string    `json:"stack,omitempty"`
	ID      uuid.UUID `json:"id,omitempty"` // Used to store output ID of a success task
}

type Task struct {
	AbortionRequestedAt APITime     `json:"abortionRequestedAt,omitempty"`
	EndedAt             APITime     `json:"end,omitempty"`
	ID                  string      `json:"id"`
	Info                DataMessage `json:"info,omitempty"`
	Properties          Properties  `json:"properties"`
	Result              Result      `json:"result,omitempty"`
	Started             APITime     `json:"start"`
	Status              Status      `json:"status"`
	UpdatedAt           APITime     `json:"updatedAt,omitempty"`
	Tasks               []Task      `json:"tasks,omitempty"`
	Warning             DataMessage `json:"warning,omitempty"`
}
