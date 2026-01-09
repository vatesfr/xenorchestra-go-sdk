# v2 Integration Tests

**TL;DR:** See [QUICK_START.md](QUICK_START.md) for quickly adding tests.

## Configuration

```bash
export XOA_URL="https://xoa.example.com"
export XOA_USER="admin"
export XOA_PASSWORD="password"
export XOA_POOL="pool-name"
export XOA_TEMPLATE="template-name"
```

Optional: `XOA_DEVELOPMENT=true`, `XOA_TEST_PREFIX`, `XOA_TEST_VLAN`

## Running Tests

```bash
make test-integration
XOA_DEVELOPMENT=true go test -v ./v2/integration/...  # With logs
go test -v -run TestName ./v2/integration/...         # Specific test
```

## Adding a Test

Template:
```go
func TestMyFeature(t *testing.T) {
    ctx, cleanup := SetupTestContext(t)
    defer cleanup()
    
    // Test code
    result, err := testClient.Pool().Get(ctx, testPool.ID)
    require.NoError(t, err)
    assert.NotNil(t, result)
}
```

Essentials:
- `SetupTestContext(t)` + `defer cleanup()` is mandatory
- Prefix resources: `integrationTestPrefix + "name"`
- Use global resources: `testClient`, `testPool`, `testTemplate`, `testNetwork`, `v1TestClient`
- `require` for fatal errors, `assert` for verifications

## Common Patterns

```go
// List with filter
vms, _ := testClient.VM().GetAll(ctx, 0, "name_label:"+integrationTestPrefix)

// Create VM
params := payloads.CreateVMParams{
    NameLabel: integrationTestPrefix + "vm",
    Template:  uuid.FromStringOrNil(testTemplate.Id),
}
vmID, _ := testClient.Pool().CreateVM(ctx, testPool.ID, params)

// Use v1 (missing resources in v2)
network, _ := v1TestClient.GetNetwork(v1.Network{Id: networkID.String()})
```

## Debugging

```bash
# Logs
XOA_DEVELOPMENT=true go test -v ./v2/integration/...

# Keep resources (temporarily comment defer cleanup())
```

---

More details: [QUICK_START.md](QUICK_START.md) or [INTEGRATION_TESTS_GUIDE.md](INTEGRATION_TESTS_GUIDE.md)
