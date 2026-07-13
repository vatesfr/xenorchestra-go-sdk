package payloads

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVM_UnmarshalJSON_TemplateFromCreation(t *testing.T) {
	templateFromField := uuid.Must(uuid.FromString("11111111-2222-3333-4444-555555555555"))
	templateFromCreation := uuid.Must(uuid.FromString("66666666-7777-8888-9999-aaaaaaaaaaaa"))

	testParams := []struct {
		name     string
		template *uuid.UUID
		creation *uuid.UUID
		expected uuid.UUID
	}{
		{"both empty", nil, nil, uuid.Nil},
		{"only creation", nil, &templateFromCreation, templateFromCreation},
		{"only template", &templateFromField, nil, templateFromField},
		{"both set", &templateFromField, &templateFromCreation, templateFromField},
	}

	for _, p := range testParams {
		t.Run(p.name, func(t *testing.T) {
			payload := []byte(fmt.Sprintf(`{
				"id": "00000000-0000-0000-0000-000000000001",
				"name_label": "vm-test",
				"power_state": "Running",
				"memory": {"size": 1024},
				"CPUs": {"number": 2},
				"type": "VM"%s%s
			}`, templateJSON(p.template), creationJSON(p.creation)))

			var vm VM
			require.NoError(t, json.Unmarshal(payload, &vm))
			assert.Equal(t, p.expected, vm.Template)
		})
	}
}

func TestCreation_UnmarshalJSON_Raw(t *testing.T) {
	const date = "2026-07-15"
	const template = "66666666-7777-8888-9999-aaaaaaaaaaaa"
	const user = "user-id"
	customKey := map[string]interface{}{"enabled": true}

	data := fmt.Appendf(nil, `{
		"date": %q,
		"template": %q,
		"user": %q,
		"customKey": {"enabled": true}
	}`, date, template, user)

	var creation Creation
	require.NoError(t, json.Unmarshal(data, &creation))

	assert.Equal(t, template, creation.Template.String())
	assert.Equal(t, date, creation.Raw["date"])
	assert.Equal(t, template, creation.Raw["template"])
	assert.Equal(t, user, creation.Raw["user"])
	assert.Equal(t, customKey, creation.Raw["customKey"])
}

func templateJSON(template *uuid.UUID) string {
	if template == nil {
		return ""
	}

	return fmt.Sprintf(`,
				"template": %q`, template.String())
}

func creationJSON(template *uuid.UUID) string {
	if template == nil {
		return ""
	}

	return fmt.Sprintf(`,
				"creation": {
					"template": %q
				}`, template.String())
}
