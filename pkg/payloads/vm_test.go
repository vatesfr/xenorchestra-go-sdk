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
			payload := fmt.Appendf(
				[]byte{},
				`{
					"name_label": "vm-test",
					"power_state": "Running",
					"memory": {"size": 1024},
					"CPUs": {"number": 2},
					"type": "VM"%s%s
				}`,
				templateJSON(p.template),
				creationJSON(p.creation),
			)

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

// The XAPI IDL declares blocked_operations as Map (operations, String): keys are
// operation names, values are free-form reason strings. The XO API serializes it
// in camelCase as "blockedOperations".
func TestVM_UnmarshalJSON_BlockedOperations(t *testing.T) {
	testParams := []struct {
		name string
		json string
		want map[VMOperation]string
	}{
		{
			name: "reason_values",
			json: `{"blockedOperations": {"destroy": "host in maintenance", "start": "true"}}`,
			want: map[VMOperation]string{VMOperationDestroy: "host in maintenance", VMOperationStart: "true"},
		},
		{
			// The value side is a free-form string, not an operation name.
			name: "value_is_not_operation",
			json: `{"blockedOperations": {"destroy": "maintenance window until Friday"}}`,
			want: map[VMOperation]string{VMOperationDestroy: "maintenance window until Friday"},
		},
		{
			// The key type is string-based, so an operation name added by a newer
			// XOS version still decodes (open enum).
			name: "unknown_operation_name",
			json: `{"blockedOperations": {"some_future_op": "why not"}}`,
			want: map[VMOperation]string{VMOperation("some_future_op"): "why not"},
		},
		{
			name: "absent_field",
			json: `{}`,
			want: nil,
		},
	}

	for _, p := range testParams {
		t.Run(p.name, func(t *testing.T) {
			var vm VM
			require.NoError(t, json.Unmarshal([]byte(p.json), &vm))
			assert.Equal(t, p.want, vm.BlockedOperations)
		})
	}
}

// The XAPI IDL declares current_operations as Map (String, operations): keys are
// running task references, values are the operation name the task performs.
func TestVM_UnmarshalJSON_CurrentOperations(t *testing.T) {
	taskA := "00000000-0000-0000-0000-0000000000aa"
	taskB := "00000000-0000-0000-0000-0000000000bb"

	testParams := []struct {
		name string
		json string
		want map[string]VMOperation
	}{
		{
			name: "single_entry",
			json: fmt.Sprintf(`{"current_operations": {%q: %q}}`, taskA, VMOperationChangingVCPUs),
			want: map[string]VMOperation{taskA: VMOperationChangingVCPUs},
		},
		{
			name: "multiple_entries",
			json: fmt.Sprintf(
				`{"current_operations": {%q: %q, %q: %q}}`,
				taskA,
				VMOperationStart,
				taskB,
				VMOperationChangingMemoryLive,
			),
			want: map[string]VMOperation{taskA: VMOperationStart, taskB: VMOperationChangingMemoryLive},
		},
		{
			// The operations enum is open: XOS versions keep adding names, so an
			// unknown value must not fail decoding.
			name: "unknown_operation_name",
			json: fmt.Sprintf(`{"current_operations": {%q: %q}}`, taskA, "some_future_op"),
			want: map[string]VMOperation{taskA: VMOperation("some_future_op")},
		},
		{
			name: "absent_field",
			json: `{}`,
			want: nil,
		},
	}

	for _, p := range testParams {
		t.Run(p.name, func(t *testing.T) {
			var vm VM
			require.NoError(t, json.Unmarshal([]byte(p.json), &vm))
			assert.Equal(t, p.want, vm.CurrentOperations)
		})
	}
}

func opsMap(ops ...VMOperation) map[string]VMOperation {
	m := make(map[string]VMOperation, len(ops))
	for i, op := range ops {
		m[fmt.Sprintf("%x", i)] = op
	}
	return m
}

// predicateResult holds the expected answers of IsShuttingDown (suspend /
// clean_shutdown / hard_shutdown), IsStarting (start / start_on / resume /
// resume_on / unpause) and IsRebooting (clean_reboot / hard_reboot).
type predicateResult struct {
	shuttingDown bool
	starting     bool
	rebooting    bool
}

// Each row is one shape of CurrentOperations; all three predicates are asserted
// per row, which keeps them in lockstep: they are independent, so a VM with
// tasks in several categories at once reports each of them.
func TestVM_OperationPredicates(t *testing.T) {
	testParams := []struct {
		name string
		ops  map[string]VMOperation
		want predicateResult
	}{
		// No current_operations field in the API payload.
		{name: "nil_map", ops: nil, want: predicateResult{}},
		{name: "empty_map", ops: map[string]VMOperation{}, want: predicateResult{}},
		{name: "suspend", ops: opsMap(VMOperationSuspend), want: predicateResult{shuttingDown: true}},
		{name: "clean_shutdown", ops: opsMap(VMOperationCleanShutdown), want: predicateResult{shuttingDown: true}},
		{name: "hard_shutdown", ops: opsMap(VMOperationHardShutdown), want: predicateResult{shuttingDown: true}},
		{name: "start", ops: opsMap(VMOperationStart), want: predicateResult{starting: true}},
		{name: "start_on", ops: opsMap(VMOperationStartOn), want: predicateResult{starting: true}},
		{name: "resume", ops: opsMap(VMOperationResume), want: predicateResult{starting: true}},
		{name: "resume_on", ops: opsMap(VMOperationResumeOn), want: predicateResult{starting: true}},
		{name: "unpause", ops: opsMap(VMOperationUnpause), want: predicateResult{starting: true}},
		{name: "clean_reboot", ops: opsMap(VMOperationCleanReboot), want: predicateResult{rebooting: true}},
		{name: "hard_reboot", ops: opsMap(VMOperationHardReboot), want: predicateResult{rebooting: true}},
		// power_state_reset is a reset, not a reboot.
		{name: "power_state_reset", ops: opsMap(VMOperationPowerStateReset), want: predicateResult{}},
		{name: "pause", ops: opsMap(VMOperationPause), want: predicateResult{}},
		{name: "changing_memory_live", ops: opsMap(VMOperationChangingMemoryLive), want: predicateResult{}},
		// The operations enum is open: a name added by a newer XOS version must
		// not affect the predicates.
		{name: "unknown_operation", ops: opsMap(VMOperation("some_future_op")), want: predicateResult{}},
	}

	for _, p := range testParams {
		vm := &VM{CurrentOperations: p.ops}
		t.Run("IsShuttingDown/"+p.name, func(t *testing.T) {
			assert.Equal(t, p.want.shuttingDown, vm.IsShuttingDown())
		})
		t.Run("IsStarting/"+p.name, func(t *testing.T) {
			assert.Equal(t, p.want.starting, vm.IsStarting())
		})
		t.Run("IsRebooting/"+p.name, func(t *testing.T) {
			assert.Equal(t, p.want.rebooting, vm.IsRebooting())
		})
	}
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
