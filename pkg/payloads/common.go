package payloads

import (
	"encoding/json"
	"strconv"

	"github.com/gofrs/uuid"
)

type CreateResponse struct {
	ID uuid.UUID `json:"id"`
}

// StringifiedInt represents an integer that is marshaled/unmarshaled as a string in JSON
// This is used for fields like VIF Device that are stringified numbers in the API
type StringifiedInt int

func (s *StringifiedInt) UnmarshalJSON(data []byte) error {
	// First try to unmarshal as a string
	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		if stringValue == "" {
			*s = 0
			return nil
		}
		intValue, err := strconv.Atoi(stringValue)
		if err != nil {
			return err
		}
		*s = StringifiedInt(intValue)
		return nil
	}

	// If string unmarshaling fails, try as a regular integer
	var intValue int
	if err := json.Unmarshal(data, &intValue); err != nil {
		return err
	}
	*s = StringifiedInt(intValue)
	return nil
}

func (s StringifiedInt) MarshalJSON() ([]byte, error) {
	// Marshal as a string representation of the integer
	return json.Marshal(strconv.Itoa(int(s)))
}
