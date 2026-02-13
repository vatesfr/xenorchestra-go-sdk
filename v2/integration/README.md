# v2 Integration Tests - Quick Start

## System Prerequisites

### Required Software

- **Go 1.21+**
- **qemu-img**: Required for VDI export/import tests

### Why qemu-img?

VDI export/import tests use `qemu-img` to:
- **Create test disk images** in various formats (RAW, VHD/VPC)
- **Verify exported disk format and size** using `qemu-img info --output=json`

If `qemu-img` is not installed, VDI-related tests will fail with:
```
exec: "qemu-img": executable file not found in $PATH
```

## Required Configuration

```bash
export XOA_URL="https://xoa.example.com"
export XOA_USER="admin"
export XOA_PASSWORD="password"
# or
export XOA_TOKEN="token"

export XOA_POOL="pool-name"
export XOA_TEMPLATE="template-name"
export XOA_STORAGE="storage-repository-name"
```

Optional:
```bash
export XOA_DEVELOPMENT="true"          # Debug logs
export XOA_TEST_PREFIX="my-prefix-"    # Custom prefix
export XOA_TEST_VLAN="1234"            # VLAN for network tests
```

## Running Tests

```bash
make test-integration                           # All tests
XOA_DEVELOPMENT=true go test -v ./v2/integration/...  # With logs
go test -v -run TestName ./v2/integration/...  # Specific test
```

## Adding a Test

1. Create function in the appropriate file (`pool_test.go`, `vms_test.go`, etc.)

2. Basic template:
```go
func TestMyFeature(t *testing.T) {
    ctx, client, prefix := SetupTestContext(t)
    
    // Your operations
    result, err := client.Pool().Get(ctx, intTests.testPool.ID)
    require.NoError(t, err)
    assert.NotNil(t, result)
}
```

3. Key points:
   - Always `ctx, client, prefix := SetupTestContext(t)`
   - Prefix resources: `prefix + "my-vm"`
   - Use: `client` (local), `intTests.testPool`, `intTests.testTemplate`, `intTests.testNetwork`
   - Unexpected errors: `require.NoError`
   - Verifications: `assert.Equal`

## Available Resources (Global)

Shared resources are available via the `intTests` global variable:

- `intTests.testPool` (payloads.Pool) - Test pool
- `intTests.testTemplate` (v1.Template) - VM template
- `intTests.testNetwork` (v1.Network) - Pool network
- `intTests.v1Client` (v1.XOClient) - v1 client for setup/teardown tasks

## Common Patterns

### Shared Helpers
Common setup logic is centralized in `helpers_test.go`:
- `createVMsForTest(t, ctx, client.Pool(), count, prefix)`: Creates multiple VMs using the test template.
- `waitForTask(t, ctx, client, taskID)`: Waits until the task with the given ID is no longer pending and returns the final task details.
- `waitForVMReady(t, ctx, client, vmID)`: Waits until the VM is in the Running state and has a MainIpAddress assigned, indicating it's ready for use.
- `createVDIForTest(t, ctx, client, name, size)`: Creates a VDI with the specified name and size using the v1 client and returns its ID.
- `createTestDiskImage(t, format, size)`: Creates a temporary disk image with qemu-img.
- `verifyDiskFormat(t, exportedContent, expectedFormat)`: Saves the exported content to a temporary file, runs qemu-img info to verify the format.

### List with filter
```go
vms, _ := client.VM().GetAll(ctx, 0, "name_label:"+prefix)
```

### Create a VM
```go
params := payloads.CreateVMParams{
    NameLabel: prefix + "vm-name",
    Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
}
vmID, _ := client.Pool().CreateVM(ctx, intTests.testPool.ID, params)
```

### Use v1 (resources not yet available in v2)
```go
network, _ := intTests.v1Client.GetNetwork(v1.Network{Id: networkID.String()})
```

## Quick Debug

```bash
# Detailed logs
XOA_DEVELOPMENT=true go test -v ./v2/integration/...

# Find the error
go test -v ./v2/integration/... 2>&1 | grep -i error
```

```golang
// Keep resources (temporarily comment defer cleanup)
// defer cleanup()
```

## Common Errors

| Error | Fix |
|-------|-----|
| "XOA_POOL must be set" | `export XOA_POOL="..."` |
| "failed to find pool" | Check exact pool name |
| "cannot connect" | Verify `XOA_URL` |
| Orphaned resources | Check `defer cleanup()` |
| `qemu-img: executable file not found` | Install qemu-img (see System Prerequisites) |

---

**See also:** [Integration Tests Guide v2](../../docs/v2/06-integration-test-guide.md) for more details.
