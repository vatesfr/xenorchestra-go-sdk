# v2 Integration Tests - Quick Start

## Required Configuration

```bash
export XOA_URL="https://xoa.example.com"
export XOA_USER="admin"
export XOA_PASSWORD="password"
# or
export XOA_TOKEN="token"

export XOA_POOL="pool-name"
export XOA_TEMPLATE="template-name"
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
    ctx, cleanup := SetupTestContext(t)
    defer cleanup()
    
    // Your operations
    result, err := testClient.Pool().Get(ctx, testPool.ID)
    require.NoError(t, err)
    assert.NotNil(t, result)
}
```

3. Key points:
   - Always `SetupTestContext(t)` + `defer cleanup()`
   - Prefix resources: `integrationTestPrefix + "my-vm"`
   - Use: `testClient`, `testPool`, `testTemplate`, `testNetwork`
   - Unexpected errors: `require.NoError`
   - Verifications: `assert.Equal`

## Available Resources (Global)

- `testClient` (library.Library) - v2 client
- `testPool` (payloads.Pool) - Test pool
- `testTemplate` (v1.Template) - VM template
- `testNetwork` (v1.Network) - Pool network
- `v1TestClient` (v1.XOClient) - v1 client for missing resources
- `integrationTestPrefix` (string) - Unique test prefix

## Common Patterns

### List with filter
```go
vms, _ := testClient.VM().GetAll(ctx, 0, "name_label:"+integrationTestPrefix)
```

### Create a VM
```go
params := payloads.CreateVMParams{
    NameLabel: integrationTestPrefix + "vm-name",
    Template:  uuid.FromStringOrNil(testTemplate.Id),
}
vmID, _ := testClient.Pool().CreateVM(ctx, testPool.ID, params)
```

### Use v1 (resources not yet available in v2)
```go
network, _ := v1TestClient.GetNetwork(v1.Network{Id: networkID.String()})
```

## Quick Debug

```bash
# Detailed logs
XOA_DEVELOPMENT=true go test -v ./v2/integration/...

# Find the error
go test -v ./v2/integration/... 2>&1 | grep -i error

# Keep resources (temporarily comment defer cleanup)
```

## Common Errors

| Error | Fix |
|-------|-----|
| "XOA_POOL must be set" | `export XOA_POOL="..."` |
| "failed to find pool" | Check exact pool name |
| "cannot connect" | Verify `XOA_URL` |
| Orphaned resources | Check `defer cleanup()` |

---

**See also:** [INTEGRATION_TESTS_GUIDE.md](INTEGRATION_TESTS_GUIDE.md) for more details.
