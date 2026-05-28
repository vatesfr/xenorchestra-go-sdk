package payloads

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringifiedInt_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    StringifiedInt
		expected string
	}{
		{"zero", 0, "\"0\""},
		{"positive", 42, "\"42\""},
		{"negative", -5, "\"-5\""},
		{"large", 999999, "\"999999\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			require.NoError(t, err, "MarshalJSON should not return an error")
			assert.Equal(t, tt.expected, string(data), "Marshaled JSON should match expected string")
		})
	}
}

func TestStringifiedInt_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected StringifiedInt
		wantErr  bool
	}{
		{"string_zero", `"0"`, 0, false},
		{"string_positive", `"42"`, 42, false},
		{"string_negative", `"-5"`, -5, false},
		{"string_large", `"999999"`, 999999, false},
		{"int_zero", `0`, 0, false},
		{"int_positive", `42`, 42, false},
		{"int_negative", `-5`, -5, false},
		{"empty_string", `""`, 0, false},
		{"invalid_string", `"abc"`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result StringifiedInt
			err := json.Unmarshal([]byte(tt.input), &result)
			require.Equal(t, tt.wantErr, err != nil, "UnmarshalJSON error presence should match wantErr")
			if !tt.wantErr {
				assert.Equal(t, tt.expected, result, "Unmarshaled value should match expected")
			}
		})
	}
}
