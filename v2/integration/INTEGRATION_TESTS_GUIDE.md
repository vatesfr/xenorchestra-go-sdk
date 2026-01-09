# Integration Tests Guide v2

This document explains how the XenOrchestra v2 SDK integration tests work and how to add new ones.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Configuration and Initialization](#configuration-and-initialization)
- [Adding a New Integration Test](#adding-a-new-integration-test)
- [Patterns and Best Practices](#patterns-and-best-practices)
- [Running Tests](#running-tests)
- [Troubleshooting](#troubleshooting)

## Overview

Integration tests test the SDK against a real XenOrchestra (XOA) instance. They verify that:

- The v2 client communicates correctly with the REST API
- CRUD operations work as expected
- Errors are handled correctly
- The v1 client lazy loading works

**Important**: These tests require an accessible XOA instance with appropriate resources (pool, template, network, etc.).

## Architecture

### Main Files

```
v2/integration/
├── setup_test.go          # Global configuration and utilities
├── client_test.go         # Client tests and lazy initialization
├── pool_test.go           # Pool operations tests (VMs, networks)
├── vms_test.go            # VM operations tests
└── logs/                  # Test logs folder
```

### Key Components

#### 1. **TestMain** (`setup_test.go`)

Single entry point for the entire integration test suite. This is where:

```go
func TestMain(m *testing.M) {
    // 1. Initialize global context
    // 2. Configure logger
    // 3. Load config from env vars
    // 4. Create v2 client
    // 5. Create v1 client (for resources not yet available in v2)
    // 6. Discover resources for tests (pool, template, network)
    // 7. Generate unique test prefix
}
```

#### 2. **Global Context and Clients**

Variables shared between all tests:

```go
var (
    integrationCtx      context.Context      // Parent context for all tests
    testClient          library.Library      // v2 client
    v1TestClient        v1.XOClient          // v1 client for missing resources
    testPool            payloads.Pool        // Test pool discovered during init
    testTemplate        v1.Template          // Template for creating VMs
    testNetwork         v1.Network           // Network for tests
    integrationTestPrefix string             // Unique prefix to identify created resources
)
```

#### 3. **SetupTestContext**

Each test must call this function to get an isolated context with timeout and cleanup:

```go
func SetupTestContext(t *testing.T) (context.Context, func()) {
    // Returns:
    // - A context with 5 minute timeout
    // - A cleanup() function that removes created resources
}
```

The cleanup:
- Deletes all VMs created with the test prefix
- Deletes all networks created with the test prefix

## Configuration and Initialization

### Required Environment Variables

| Variable | Description | Example | Required |
|----------|-------------|---------|----------|
| `XOA_URL` | URL of the XOA API | `https://xoa.example.com` | ✅ |
| `XOA_USER` | Username | `admin` | If no token |
| `XOA_PASSWORD` | Password | `password123` | If no token |
| `XOA_TOKEN` | Authentication token | `xxxxxxx` | If no credentials |
| `XOA_POOL` | Test pool name | `pool-1` | ✅ |
| `XOA_TEMPLATE` | Template name for VMs | `Debian` | ✅ |
| `XOA_DEVELOPMENT` | Enable debug logs | `true` | ❌ |
| `XOA_TEST_PREFIX` | Custom test prefix | `my-test-` | ❌ |
| `XOA_TEST_VLAN` | VLAN for network tests | `1234` | ❌ |

### Configuration Example

```bash
export XOA_URL="https://xoa.lab.local"
export XOA_USER="admin"
export XOA_PASSWORD="admin"
export XOA_POOL="local"
export XOA_TEMPLATE="Debian 11"
export XOA_DEVELOPMENT="true"  # For detailed logs
export XOA_TEST_PREFIX="integration-test-"
```

### Initialization Flow

1. **TestMain runs once** at the start of the test suite
2. **Configuration loaded** from env vars via `config.New()`
3. **v2 client created** for REST operations
4. **v1 client created** for resources not yet available in v2
5. **Pool discovered** by name (env var `XOA_POOL`)
6. **Template discovered** by name (env var `XOA_TEMPLATE`)
7. **Network discovered** from the pool (first PIF)
8. **Unique prefix generated** with timestamp to identify test resources

## Adding a New Integration Test

### Step 1: Create the Appropriate Test File

Place your test in the corresponding file based on the resource being tested:

- **Pool** operations → `pool_test.go`
- **VM** operations → `vms_test.go`
- **Client** operations → `client_test.go`
- Other → create `resource_test.go`

### Step 2: Write Your Test

Here's a complete example to test VM creation and deletion:

```go
func TestCreateAndDeleteVM(t *testing.T) {
    // 1. Setup: get isolated context and cleanup
    ctx, cleanup := SetupTestContext(t)
    defer cleanup()  // Cleanup automatically removes the VM

    // 2. Prepare parameters
    vmName := integrationTestPrefix + "my-test-vm"
    params := payloads.CreateVMParams{
        NameLabel: vmName,
        Template:  uuid.FromStringOrNil(testTemplate.Id),
    }

    // 3. Execute operation
    vmID, err := testClient.Pool().CreateVM(ctx, testPool.ID, params)
    require.NoError(t, err, "VM creation failed")
    require.NotEqual(t, uuid.Nil, vmID, "invalid VM ID")

    // 4. Verify result
    vm, err := testClient.VM().Get(ctx, vmID)
    require.NoError(t, err, "VM retrieval failed")
    assert.Equal(t, vmName, vm.NameLabel, "VM name incorrect")
    assert.Equal(t, testPool.ID, vm.PoolId, "pool ID incorrect")
}
```

### Step 3: Use Utilities

#### Helper: `createVMsForTest`

To quickly create multiple VMs:

```go
func TestVMBatch(t *testing.T) {
    ctx, cleanup := SetupTestContext(t)
    defer cleanup()

    // Creates 5 VMs with unique names
    vmIDs := createVMsForTest(t, ctx, 5, "batch-test-")
    assert.Len(t, vmIDs, 5)
}
```

#### Helper: `cleanupVMsWithPrefix`

To manually clean VMs (normally managed by SetupTestContext):

```go
err := cleanupVMsWithPrefix(integrationTestPrefix + "my-prefix-")
require.NoError(t, err)
```

### Step 4: Test with v1 APIs if Necessary

For resources not yet available in v2, use the v1 client:

```go
func TestNetworkCreation(t *testing.T) {
    ctx, cleanup := SetupTestContext(t)
    defer cleanup()

    // Create with v2
    networkID, err := testClient.Pool().CreateNetwork(ctx, testPool.ID, params)
    require.NoError(t, err)

    // Verify with v1 (Network service v2 doesn't exist yet)
    network, err := v1TestClient.GetNetwork(v1.Network{
        Id: networkID.String(),
    })
    require.NoError(t, err)
    assert.Equal(t, params.Name, network.NameLabel)
}
```

## Patterns and Best Practices

### 1. Always Call SetupTestContext

```go
// ✅ GOOD
func TestSomething(t *testing.T) {
    ctx, cleanup := SetupTestContext(t)
    defer cleanup()
    // test code
}

// ❌ BAD
func TestSomething(t *testing.T) {
    // No context, manual cleanup easily forgotten
    vm, _ := testClient.VM().Get(integrationCtx, vmID)
}
```

### 2. Use Prefix for Resource Names

```go
// ✅ GOOD - easily identifies test resources
vmName := integrationTestPrefix + "my-vm"

// ❌ BAD - hard to filter in case of problems
vmName := "my-vm"
```

### 3. Check Errors with require

```go
// ✅ GOOD - stops at first error, more readable
vmID, err := testClient.Pool().CreateVM(ctx, testPool.ID, params)
require.NoError(t, err, "VM creation failed")

// ⚠️ OK but less clear
vmID, err := testClient.Pool().CreateVM(ctx, testPool.ID, params)
if err != nil {
    t.Fatalf("VM creation failed: %v", err)
}
```

### 4. Add Descriptive Messages

```go
// ✅ GOOD
assert.Equal(t, expected, actual, "VM name should match")

// ❌ BAD
assert.Equal(t, expected, actual)
```

### 5. Explicitly Test Errors

```go
func TestCreateVMWithInvalidTemplate(t *testing.T) {
    ctx, cleanup := SetupTestContext(t)
    defer cleanup()

    params := payloads.CreateVMParams{
        NameLabel: integrationTestPrefix + "invalid-template",
        Template:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
    }

    _, err := testClient.Pool().CreateVM(ctx, testPool.ID, params)
    
    // Verify expected error
    require.Error(t, err, "VM creation with invalid template should fail")
    assert.Contains(t, err.Error(), "404", "error should mention 404")
}
```

### 6. Organize Tests as Sub-tests

```go
func TestVMOperations(t *testing.T) {
    ctx, cleanup := SetupTestContext(t)
    defer cleanup()

    // Common setup
    vm := createTestVM(t, ctx)

    // Sub-tests
    t.Run("Get", func(t *testing.T) {
        retrieved, err := testClient.VM().Get(ctx, vm.ID)
        require.NoError(t, err)
        assert.Equal(t, vm.ID, retrieved.ID)
    })

    t.Run("Update", func(t *testing.T) {
        // ...
    })

    t.Run("Delete", func(t *testing.T) {
        err := testClient.VM().Delete(ctx, vm.ID)
        require.NoError(t, err)
    })
}
```

## Running Tests

### All Tests (with v1)

```bash
make test
```

### Only v2 Tests (without integration)

```bash
make test-v2
```

### Only Integration Tests

```bash
make test-integration
```

With more details (verbose):

```bash
make test-integration GOFLAGS="-v"
```

With debug logs:

```bash
XOA_DEVELOPMENT=true make test-integration GOFLAGS="-v"
```

### Single Test

```bash
go test -v -run TestCreateVM ./v2/integration/...
```

### With Custom Timeout

```bash
go test -timeout 10m ./v2/integration/...
```

### With Custom Prefix to Avoid Collisions

```bash
XOA_TEST_PREFIX="ci-run-123-" go test -v ./v2/integration/...
```

## Troubleshooting

### Problem: "The XOA_POOL environment variable must be set"

**Cause**: Missing environment variable

**Solution**:
```bash
export XOA_POOL="name-of-your-pool"
```

### Problem: "failed to get pool with name"

**Cause**: Exact pool name not found

**Solution**:
1. Check the exact name in XOA
2. Ensure only one pool has this name
3. Check access permissions

### Problem: "failed to create client"

**Cause**: Invalid authentication or URL

**Solution**:
```bash
# Check credentials
export XOA_URL="https://xoa.example.com"
export XOA_USER="admin"
export XOA_PASSWORD="correct-password"
# Or use a token
export XOA_TOKEN="your-token-here"
```

### Problem: Tests Leaving Orphaned VMs/Networks

**Cause**: Cleanup couldn't complete (timeout, network error)

**Solution**:
1. Clean up manually from XOA
2. Use a unique prefix to identify abandoned resources
3. Check logs: `tail -f v2/integration/logs/*.log`

### Problem: Timeout During Tests

**Cause**: XOA too slow or slow network

**Solution**:
```bash
# Increase timeout
go test -timeout 10m ./v2/integration/...
```

### Problem: "v1Client should not be initialized yet (should be nil)"

**Cause**: v1 client was initialized earlier than expected

**Solution**:
- Check that you don't access `v1TestClient` or `client.V1Client()` during TestMain
- Lazy loading is specifically tested in `client_test.go`

## Complete Example

Here's a complete example of adding a new integration test for a hypothetical "VM reboot" operation:

```go
// vm_reboot_test.go
package integration

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/gofrs/uuid"
    "github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestVMReboot(t *testing.T) {
    ctx, cleanup := SetupTestContext(t)
    defer cleanup()

    // Create a VM for the test
    vmName := integrationTestPrefix + "reboot-test-vm"
    params := payloads.CreateVMParams{
        NameLabel: vmName,
        Template:  uuid.FromStringOrNil(testTemplate.Id),
    }

    vmID, err := testClient.Pool().CreateVM(ctx, testPool.ID, params)
    require.NoError(t, err, "VM creation failed")

    // Verify VM exists
    vm, err := testClient.VM().Get(ctx, vmID)
    require.NoError(t, err, "VM retrieval failed")
    assert.Equal(t, vmName, vm.NameLabel)

    // Test reboot (once this operation is available)
    // err = testClient.VM().Reboot(ctx, vmID)
    // require.NoError(t, err, "reboot failed")

    // Verify state after reboot
    // vm, err = testClient.VM().Get(ctx, vmID)
    // require.NoError(t, err)
    // assert.True(t, vm.Running)
}

// Sub-tests for error scenarios
func TestVMRebootErrors(t *testing.T) {
    ctx, cleanup := SetupTestContext(t)
    defer cleanup()

    t.Run("InvalidVM", func(t *testing.T) {
        // err := testClient.VM().Reboot(ctx, uuid.Nil)
        // require.Error(t, err, "reboot of invalid VM should fail")
    })
}
```

## Additional Resources

- v2 documentation: see `docs/v2/` in the repository
- v2 examples: see `examples/v2/` in the repository
- XO API documentation: https://xen-orchestra.com/docs/api.html
