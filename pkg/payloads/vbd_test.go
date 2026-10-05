package payloads

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVBDUnmarshalJSON(t *testing.T) {
	t.Run("numeric position is kept verbatim", func(t *testing.T) {
		var vbd VBD
		require.NoError(t, json.Unmarshal([]byte(`{
			"id": "5f9a4c0b-3548-6171-4396-699a0e56cc60",
			"uuid": "5f9a4c0b-3548-6171-4396-699a0e56cc60",
			"type": "VBD",
			"device": "xvda",
			"position": "0",
			"VM": "4fe90510-8da4-1530-38e2-a7876ef374c7"
		}`), &vbd))

		assert.Equal(t, "0", vbd.Position)
		assert.Equal(t, uuid.Must(uuid.FromString("4fe90510-8da4-1530-38e2-a7876ef374c7")), vbd.VM)
	})

	// Regression: the REST API exposes the XAPI "userdevice" under the
	// "position" key and its content is data dependent: some stacks return a
	// numeric index ("0"), others a device name ("xvdb", "cd0"). The field
	// must not be coerced to a number, or unmarshaling fails for the whole
	// VBD listing.
	t.Run("device-name position does not fail unmarshaling", func(t *testing.T) {
		var vbd VBD
		require.NoError(t, json.Unmarshal([]byte(`{
			"id": "5f9a4c0b-3548-6171-4396-699a0e56cc60",
			"uuid": "5f9a4c0b-3548-6171-4396-699a0e56cc60",
			"type": "VBD",
			"device": "xvdb",
			"position": "xvdb",
			"VM": "4fe90510-8da4-1530-38e2-a7876ef374c7"
		}`), &vbd))

		assert.Equal(t, "xvdb", vbd.Position)
	})

	t.Run("missing position is an empty string", func(t *testing.T) {
		var vbd VBD
		require.NoError(t, json.Unmarshal([]byte(`{"type": "VBD"}`), &vbd))

		assert.Equal(t, "", vbd.Position)
	})
}
