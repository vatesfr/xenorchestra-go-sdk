# Integration Tests Guide v2

This document explains how the Xen Orchestra v2 SDK integration tests work and how to add new ones.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Configuration and Initialization](#configuration-and-initialization)
- [Adding a New Integration Test](#adding-a-new-integration-test)
- [Patterns and Best Practices](#patterns-and-best-practices)
- [Running Tests](#running-tests)
- [Troubleshooting](#troubleshooting)

## Overview

Integration tests verify the SDK against a real Xen Orchestra instance. They ensure:

- The v2 client communicates correctly with the REST API.
- CRUD operations work as expected.
- Errors are handled correctly (e.g., 404 Not Found).
- Resources not yet available in v2 can still be managed via the v1 client.

**Important**: These tests require an accessible XOA instance with appropriate resources (pool, template, network, etc.).

## Architecture

### Key Components

#### 1. **TestMain** (`setup_test.go`)

The entry point for the integration test suite. It performs global initialization:

```go
func TestMain(m *testing.M) {
    // 1. Initialize global integrationTestContext
    // 2. Configure slog logger (respects XOA_DEVELOPMENT)
    // 3. Load config from env vars (XOA_URL, etc.)
    // 4. Initialize v1 client for setup/teardown tasks
    // 5. Discover resources (testPool, testNetwork, testTemplate)
    // 6. Generate global testPrefix (includes timestamp)
}
```

#### 2. **Global Integration Context**

Shared resources are stored in the `intTests` global variable:

```go
type integrationTestContext struct {
    ctx          context.Context // Parent context
    testConfig   *config.Config  // SDK configuration
    testPool     payloads.Pool   // Pool used for testing (v2)
    testTemplate v1.Template     // Template used for VM creation (v1)
    testNetwork  v1.Network      // Network used for tests (v1)
    v1Client     v1.XOClient     // Client for missing v2 features
}
```

#### 3. **SetupTestContext**

Every test must call `SetupTestContext` at the beginning. It prepares an isolated environment and automatically registers a cleanup function via `t.Cleanup()`:

```go
func SetupTestContext(t *testing.T) (context.Context, library.Library, string) {
    // Returns:
    // - ctx: A context with a 5-minute timeout.
    // - client: A fresh v2 SDK client.
    // - prefix: A unique name prefix for this specific test (e.g., "xo-go-sdk-12345-TestMyFeature-").
}
```

#### 4. **Test Helpers** (`helpers_test.go`)

Common utility functions used across multiple test files are centralized in `helpers_test.go`. These helpers simplify setup tasks like creating multiple VMs or finding resources.

## Configuration and Initialization

### Required Environment Variables

| Variable | Description | Example | Required |
|----------|-------------|---------|----------|
| `XOA_URL` | URL of the XOA API | `https://xoa.example.com` | ✅ |
| `XOA_USER` | Username | `admin` | If no token |
| `XOA_PASSWORD` | Password | `password123` | If no token |
| `XOA_TOKEN` | Authentication token | `xxxxxxx` | If no credentials |
| `XOA_POOL` | Test pool Name Label | `My Pool` | ✅ |
| `XOA_TEMPLATE` | Template Name Label | `Debian 12` | ✅ |
| `XOA_DEVELOPMENT`| Enable debug logs | `true` | ❌ |
| `XOA_TEST_PREFIX`| Custom resource prefix | `ci-` | ❌ |
| `XOA_TEST_VLAN`  | VLAN for network tests | `1234` | ❌ |

### Running the Suite

```bash
export XOA_URL="https://xoa.lab.local"
export XOA_TOKEN="token"
export XOA_POOL="Pool A"
export XOA_TEMPLATE="Debian 12"

make test-integration
```

## Adding a New Integration Test

### Step 1: Choose or Create a Test File

Organize tests by resource type in `v2/integration/`:
- `pool_test.go`: Pool and resource creation (VMs, Networks).
- `vms_test.go`: VM management and listing.
- `client_test.go`: SDK client behavior (lazy loading, etc.).

### Step 2: Implement the Test

Always use `SetupTestContext` and the provided `prefix` for resource names to ensure safe cleanup and avoid collisions.

```go
func TestCreateVM(t *testing.T) {
    // 1. Setup environment (cleanup is automatically registered)
    ctx, client, prefix := SetupTestContext(t)

    // 2. Prepare parameters using global resources
    vmName := prefix + "my-vm"
    params := &payloads.CreateVMParams{
        NameLabel: vmName,
        Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
    }

    // 3. Execute v2 operation
    vmID, err := client.VM().Create(ctx, intTests.testPool.ID, params)
    require.NoError(t, err)
    assert.NotEqual(t, uuid.Nil, vmID)

    // 4. Verification
    vm, err := client.VM().GetByID(ctx, vmID)
    require.NoError(t, err)
    assert.Equal(t, vmName, vm.NameLabel)
}
```

### Step 3: Use Helpers

#### `createVMsForTest`
Located in `helpers_test.go`, it helps create multiple VMs for listing or batch tests:
```go
vmsIDs := createVMsForTest(t, ctx, client.Pool(), 3, prefix + "batch-")
```

If you find yourself repeating setup logic in multiple test files, add a new helper function to `helpers_test.go`. Ensure it:
1.  Uses `*testing.T` for assertions (`require.NoError`, etc.).
2.  Accepts `context.Context` and the necessary clients.
3.  Uses the provided `prefix` for resource names to ensure safe cleanup.

## Patterns and Best Practices

1.  **Isolated Cleanup**: `SetupTestContext` registers a `t.Cleanup()` function that deletes resources starting with that specific test's `prefix`. This allows tests to run without interfering with each other, even in parallel.
2.  **Global vs. Local Client**: Use the `client` returned by `SetupTestContext` for the actual test. Use `intTests.v1Client` only for setup or verification tasks where v2 is not yet implemented.
3.  **Error Assertion**: Use `require.NoError` for setup steps to fail fast, and `assert` for the actual test conditions.
4.  **Descriptive Assertions**: Include messages in assertions: `assert.Equal(t, expected, actual, "VM name should match")`.
5.  **Subtests and Parallelism**: When using `t.Run`, you can also call `SetupTestContext(t)` inside the subtest to get a fresh context and automatic cleanup specific to that subtest. Use `t.Parallel()` to speed up execution.

## Running Tests

### All Integration Tests
```bash
make test-integration
```

### Single Test File or Function
```bash
# Run only VM tests
go test -v ./v2/integration/vms_test.go ./v2/integration/setup_test.go ./v2/integration/helpers_test.go 

# Run a specific test function
go test -v -run TestVmCreation ./v2/integration/...
```

### With Debug Logs
```bash
XOA_DEVELOPMENT=true make test-integration
```

## Troubleshooting

- **"XOA_POOL variable must be set"**: Ensure you have exported all required variables.
- **Found X pools with name_label**: The pool name must be unique in your XOA instance.
- **Orphaned Resources**: If a test is forcefully killed, resources might remain. They will all start with the prefix (e.g., `xo-go-sdk-`). You can manually delete them or run a test that triggers cleanup.
- **v1/v2 mismatches**: If you see `v1.Template` or `v1.Network` in code, it's because those resource payloads aren't fully migrated to v2 yet. Use them via `intTests`.

## Additional Resources

- `v2/integration/README.md`: Quick reference for environment setup and basic template.
- `v2/integration/setup_test.go`: Implementation details of the test runner.
