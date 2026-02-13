package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
)

// createVMsForTest helps create multiple VMs for listing or batch tests
func createVMsForTest(t *testing.T, ctx context.Context, pool library.Pool, count int, name string) []string {
	vmIDs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		vmName := name + uuid.Must(uuid.NewV4()).String()
		params := payloads.CreateVMParams{
			NameLabel: vmName,
			Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		}

		vmID, err := pool.CreateVM(ctx, intTests.testPool.ID, params)
		require.NoErrorf(t, err, "error while creating VM %s in pool %s: %v", vmName, intTests.testPool.ID, err)
		require.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")
		vmIDs = append(vmIDs, vmID.String())
	}

	return vmIDs
}

func ptr[T any](v T) *T {
	return &v
}

// waitForTask waits until the task with the given ID is no longer pending and returns the final task details.
func waitForTask(t *testing.T, ctx context.Context, client library.Library, taskID string) *payloads.Task {
	t.Helper()
	var task *payloads.Task
	assert.Eventually(t, func() bool {
		var err error
		task, err = client.Task().Get(ctx, taskID)
		if err != nil {
			return false
		}
		return task.Status != payloads.Pending
	}, 2*time.Minute, 5*time.Second, "Task %s should not stay pending", taskID)
	return task
}

// waitForVMReady waits until the VM is in the Running state and has a MainIpAddress assigned,
// indicating it's ready for use.
func waitForVMReady(t *testing.T, ctx context.Context, client library.Library, vmID uuid.UUID) {
	t.Helper()
	assert.Eventually(t, func() bool {
		vm, err := client.VM().GetByID(ctx, vmID)
		if err != nil {
			return false
		}
		return vm.PowerState == payloads.PowerStateRunning && vm.MainIpAddress != ""
	}, 2*time.Minute, 10*time.Second, "VM %s should be running and reported an IP", vmID)
}

// createVDIForTest creates a VDI with the specified name and size using the v1 client and returns its ID
// TODO: replace with v2 client once VDI creation is supported in v2
func createVDIForTest(t *testing.T, ctx context.Context, client v1.XOClient, name string, size int64) uuid.UUID {
	t.Helper()

	var id string

	if client, ok := intTests.v1Client.(*v1.Client); ok {
		err := client.Call("disk.create", map[string]any{
			"name": name,
			"size": size,
			"sr":   intTests.testSR.Id,
		}, &id)
		require.NoError(t, err, "error while creating VDI %s in SR %s: %v", name, intTests.testSR.Id, err)
	}
	return uuid.FromStringOrNil(id)
}

// verifyDiskFormat saves the exported content to a temporary file, runs qemu-img info to verify the format
// comparing them against the expected values.
func verifyDiskFormat(t *testing.T, exportedContent io.ReadCloser, expectedFormat string) {
	t.Helper()

	// Create a temporary file to save the exported content
	tmpFile, err := os.CreateTemp("", "vdi-export-*.img")
	require.NoError(t, err, "creating temporary file should succeed")
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy the exported content to the temporary file
	_, err = io.Copy(tmpFile, exportedContent)
	require.NoError(t, err, "copying exported content to file should succeed")

	// Close the file to ensure all data is flushed
	err = tmpFile.Close()
	require.NoError(t, err, "closing temporary file should succeed")
	// Store the file path in a variable to satisfy gosec
	tmpPath := tmpFile.Name()

	// Run qemu-img info to get the format and size
	cmd := exec.Command("qemu-img", "info", "--output=json", tmpPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "qemu-img info should succeed: %s", string(output))

	// Parse the JSON output
	var info struct {
		Format string `json:"format"`
	}
	err = json.Unmarshal(output, &info)
	require.NoError(t, err, "parsing qemu-img info output should succeed")

	// Verify the format
	assert.Equal(t, expectedFormat, info.Format, "disk format should match expected format")
}

// createTestDiskImage creates a temporary disk image with qemu-img for import tests.
// Returns the path to the created image file. The caller is responsible for cleanup.
func createTestDiskImage(t *testing.T, format string, size int64) string {
	t.Helper()

	// Create temporary file with appropriate extension
	ext := ".img"
	if format == "vpc" {
		ext = ".vhd"
	}

	tmpFile, err := os.CreateTemp("", "test-disk-*"+ext)
	require.NoError(t, err, "creating temporary file should succeed")
	tmpFile.Close()
	// Store the file path in a variable to satisfy gosec
	tmpPath := tmpFile.Name()

	// Create disk image with qemu-img
	cmd := exec.Command("qemu-img", "create", "-f", format, tmpPath, fmt.Sprintf("%d", size))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "qemu-img create should succeed: %s", string(output))

	return tmpPath
}
