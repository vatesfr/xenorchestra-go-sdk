package payloads

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVIFParams_DeviceField(t *testing.T) {
	// Test that VIFParams can handle stringified device numbers
	jsonData := `{
		"device": "2"
	}`

	var vif VIFParams
	err := json.Unmarshal([]byte(jsonData), &vif)
	require.NoError(t, err, "Failed to unmarshal VIFParams with stringified device")
	require.NotNil(t, vif.Device, "Device field should not be nil after unmarshaling")

	expected := StringifiedInt(2)
	require.Equal(t, expected, *vif.Device, "Device field should be correctly parsed from stringified JSON")

	// Test marshaling back to JSON
	marshaled, err := json.Marshal(vif)
	require.NoError(t, err, "Failed to marshal VIFParams with stringified device")

	assert.Contains(t, string(marshaled), "\"2\"", "Marshaled JSON should contain stringified device number")
}
