/*
Those struct got some UnmarshalJSON methods to handle the different formats
of the API responses. We can expect with the switch from JS to TS that the
format will be more consistent and the need for those methods will be reduced.
*/
package payloads

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

type Status string

const (
	Success Status = "success"
	Failure Status = "failure"
	Running Status = "running"
	Pending Status = "pending"
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

type Params struct {
	ID string `json:"id"`
}

type Properties struct {
	Name     string `json:"name"`
	Method   string `json:"method,omitempty"`
	Params   Params `json:"params,omitempty"`
	ObjectID string `json:"objectId"`
	UserID   string `json:"userId"`
	Type     string `json:"type,omitempty"`
}

type Result struct {
	Code   string `json:"code"`
	Params []any  `json:"params"`
	Call   struct {
		Method   string `json:"method"`
		Duration int64  `json:"duration"`
		Params   []any  `json:"params"`
	} `json:"call"`
}

type TaskResult struct {
	ID       uuid.UUID `json:"id,omitempty"`
	Result   Result    `json:"result,omitempty"`
	StringID string
}

func (r *TaskResult) UnmarshalJSON(data []byte) error {
	var idStr string
	if err := json.Unmarshal(data, &idStr); err == nil {
		id, err := uuid.FromString(idStr)
		if err == nil {
			r.ID = id
		}
		r.StringID = idStr
		return nil
	}

	var structResult struct {
		ID     uuid.UUID `json:"id,omitempty"`
		Result Result    `json:"result,omitempty"`
	}

	if err := json.Unmarshal(data, &structResult); err != nil {
		return fmt.Errorf("value is neither a valid result string nor a structured result: %v", err)
	}

	r.ID = structResult.ID
	r.Result = structResult.Result
	return nil
}

func (r TaskResult) MarshalJSON() ([]byte, error) {
	if r.StringID != "" && r.ID == uuid.Nil {
		return json.Marshal(r.StringID)
	}

	structResult := struct {
		ID     uuid.UUID `json:"id,omitempty"`
		Result Result    `json:"result,omitempty"`
	}{
		ID:     r.ID,
		Result: r.Result,
	}

	return json.Marshal(structResult)
}

type Task struct {
	ID         string     `json:"id"`
	Name       string     `json:"name,omitempty"`
	Status     Status     `json:"status"`
	Properties Properties `json:"properties"`
	Started    APITime    `json:"start"`
	UpdatedAt  APITime    `json:"updatedAt"`
	EndedAt    APITime    `json:"end,omitempty"`
	Result     TaskResult `json:"result,omitempty"`
	Message    string     `json:"message,omitempty"`
	Stack      string     `json:"stack,omitempty"`
}
