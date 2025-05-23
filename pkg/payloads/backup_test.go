package payloads

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupJob_ToJSONRPCPayload(t *testing.T) {
	jobID := uuid.Must(uuid.NewV4())
	scheduleID := uuid.Must(uuid.NewV4())
	remoteID := uuid.Must(uuid.NewV4())
	vmID := uuid.Must(uuid.NewV4())

	remoteIDStr := remoteID.String()
	vmIDStr := vmID.String()

	compression := "zstd"
	reportWhen := ReportWhenFailOnly
	timezone := "Europe/Paris"
	backupReportTpl := "compactMjml"

	job := &BackupJob{
		ID:          jobID,
		Name:        "BACKUP-TEST",
		Mode:        BackupJobTypeFull,
		Type:        BackupJobModeBackup,
		Schedule:    scheduleID,
		Compression: &compression,
		VMs:         vmIDStr,
		Remotes:     remoteIDStr,
		Settings: BackupSettings{
			ReportWhen:       &reportWhen,
			ReportRecipients: []string{"dummy-example@example.com"},
			LongTermRetention: LongTermRetentionObject{
				"daily": LongTermRetentionDuration{
					Retention: 1,
					Settings:  map[string]any{},
				},
				"weekly": LongTermRetentionDuration{
					Retention: 1,
					Settings:  map[string]any{},
				},
				"monthly": LongTermRetentionDuration{
					Retention: 1,
					Settings:  map[string]any{},
				},
				"yearly": LongTermRetentionDuration{
					Retention: 2,
					Settings:  map[string]any{},
				},
			},
			NRetriesVmBackupFailures:  intPtr(1),
			Timeout:                   intPtr(3600000),
			MaxExportRate:             intPtr(1048576),
			OfflineBackup:             boolPtr(true),
			BackupReportTpl:           &backupReportTpl,
			MergeBackupsSynchronously: boolPtr(true),
			Timezone:                  &timezone,
			ExportRetention:           intPtr(1),
			DeleteFirst:               boolPtr(true),
		},
	}

	result := job.ToJSONRPCPayload()

	// Verify basic fields
	assert.Equal(t, "BACKUP-TEST", result["name"])
	assert.Equal(t, "full", result["mode"])
	assert.Equal(t, "backup", result["type"])
	assert.Equal(t, "zstd", result["compression"])
	assert.Equal(t, jobID.String(), result["id"])

	expectedVMs := map[string]any{
		"id": vmIDStr,
	}
	assert.Equal(t, expectedVMs, result["vms"])

	expectedRemotes := map[string]any{
		"id": remoteIDStr,
	}
	assert.Equal(t, expectedRemotes, result["remotes"])

	expectedSRs := map[string]any{
		"id": map[string]any{
			"__or": []string{},
		},
	}
	assert.Equal(t, expectedSRs, result["srs"])

	settings, ok := result["settings"].(map[string]any)
	require.True(t, ok, "settings should be a map")

	defaultSettings, ok := settings[""].(map[string]any)
	require.True(t, ok, "default settings should exist")

	assert.NotContains(t, defaultSettings, "type", "type should be at root level, not in default settings")
	assert.Equal(t, "failure", defaultSettings["reportWhen"])
	assert.Equal(t, []string{"dummy-example@example.com"}, defaultSettings["reportRecipients"])
	assert.Equal(t, true, defaultSettings["offlineBackup"])
	assert.Equal(t, "compactMjml", defaultSettings["backupReportTpl"])
	assert.Equal(t, true, defaultSettings["mergeBackupsSynchronously"])
	assert.Equal(t, "Europe/Paris", defaultSettings["timezone"])
	assert.Equal(t, 1, defaultSettings["nRetriesVmBackupFailures"])
	assert.Equal(t, 3600000, defaultSettings["timeout"])
	assert.Equal(t, 1048576, defaultSettings["maxExportRate"])

	ltr, ok := defaultSettings["longTermRetention"].(LongTermRetentionObject)
	require.True(t, ok, "longTermRetention should be present")
	assert.Equal(t, 1, ltr["daily"].Retention)
	assert.Equal(t, 1, ltr["weekly"].Retention)
	assert.Equal(t, 1, ltr["monthly"].Retention)
	assert.Equal(t, 2, ltr["yearly"].Retention)

	scheduleSettings, ok := settings[scheduleID.String()].(map[string]any)
	require.True(t, ok, "schedule-specific settings should exist")
	assert.Equal(t, 1, scheduleSettings["exportRetention"])
	assert.Len(t, scheduleSettings, 1, "schedule settings should only contain exportRetention")

	remoteSettings, ok := settings[remoteIDStr].(map[string]any)
	require.True(t, ok, "remote-specific settings should exist")
	assert.Equal(t, true, remoteSettings["deleteFirst"])
	assert.Len(t, remoteSettings, 1, "remote settings should only contain deleteFirst")

	assert.Len(t, settings, 3, "should have exactly 3 settings blocks: default, schedule, and remote")
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
