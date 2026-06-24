package integration

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	v2 "github.com/vatesfr/xenorchestra-go-sdk/v2"
)

type integrationTestContext struct {
	// integrationCtx is the parent context for all tests
	ctx context.Context
	// Client is the XO v2 client used for testing
	// Client library.Library
	testConfig *config.Config

	// testPool holds the pool used for testing
	testPool payloads.Pool

	// testSR holds a storage repository used for VDI-related tests
	testSR payloads.StorageRepository

	// testTemplateID holds the template UUID used for VM creation tests.
	// Resolved from XOA_TEMPLATE_ID (direct) or v1 discovery (fallback).
	testTemplateID string

	// testNetworkID holds the network UUID used for network-related tests.
	// Resolved from XOA_NETWORK_ID (direct) or v1 discovery (fallback).
	testNetworkID string

	// v1Disabled is true when XOA_DISABLE_V1=true.
	// When true, v1Client is nil and v1-dependent tests are skipped.
	v1Disabled bool

	// v1Client is the XO client used for resources not yet available in v2.
	// Should not be used to perform the actual test but only to setup/teardown resources.
	// Nil when v1Disabled is true.
	v1Client v1.XOClient

	// testPBD is the UUID of a PBD that is safe to temporarily plug/unplug during tests.
	// Populated from XOA_TEST_PBD_ID. When uuid.Nil, plug/unplug tests are skipped.
	testPBD uuid.UUID
}

var (
	// intTests holds global test configuration and resources shared across all integration tests
	intTests       integrationTestContext = integrationTestContext{}
	intTestsPrefix string                 = "xo-go-sdk-"
)

// TestMain is the main entry point for integration tests
func TestMain(m *testing.M) {
	var err error
	var integrationCancel context.CancelFunc
	// Global setup
	intTests.ctx, integrationCancel = context.WithCancel(context.Background())

	devMode, _ := strconv.ParseBool(os.Getenv("XOA_DEVELOPMENT"))

	// Create logger
	handlerOpt := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	// Use development mode for tests
	if devMode {
		handlerOpt.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, handlerOpt))
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// XO client configuration via environment variables
	// - XOA_URL: XO API URL (required)
	// - XOA_USER and XOA_PASSWORD: Credentials (required if no token)
	// - XOA_TOKEN: Authentication token (required if no credentials)
	// - XOA_DEVELOPMENT: true to enable development logs
	// - XOA_DISABLE_V1: true to disable v1 client (requires XOA_TEMPLATE_ID and XOA_NETWORK_ID)
	// - XOA_TEMPLATE_ID: direct template UUID (takes precedence over v1 discovery)
	// - XOA_NETWORK_ID: direct network UUID (takes precedence over v1 discovery)
	intTests.testConfig, err = config.New()
	if err != nil {
		log.Fatalf("configuration failed: %v", err)
	}

	// Force development mode for tests
	intTests.testConfig.Development = devMode

	// Determine v1 status
	intTests.v1Disabled, _ = strconv.ParseBool(os.Getenv("XOA_DISABLE_V1"))

	// Resolve template ID: direct env var takes precedence, fallback to v1 discovery
	if templateID, found := os.LookupEnv("XOA_TEMPLATE_ID"); found && templateID != "" {
		intTests.testTemplateID = templateID
	}

	// Resolve network ID: direct env var takes precedence, fallback to v1 discovery
	if networkID, found := os.LookupEnv("XOA_NETWORK_ID"); found && networkID != "" {
		intTests.testNetworkID = networkID
	}

	// When v1 is disabled, both direct IDs must be provided
	if intTests.v1Disabled {
		if intTests.testTemplateID == "" {
			integrationCancel()
			log.Fatal("XOA_DISABLE_V1=true requires XOA_TEMPLATE_ID to be set")
		}
		if intTests.testNetworkID == "" {
			integrationCancel()
			log.Fatal("XOA_DISABLE_V1=true requires XOA_NETWORK_ID to be set")
		}
	}

	// Get information for testing
	intTests.testPool = findPoolForTests()
	intTests.testSR = findStorageRepositoryForTests()
	intTests.testPBD = findPBDForTests()

	// Initialize v1 client only when needed for discovery or v1-dependent teardown
	if !intTests.v1Disabled {
		intTests.v1Client, err = v1.NewClientWithLogger(v1.GetConfigFromEnv(), logger)
		if err != nil {
			integrationCancel()
			log.Fatalf("error getting v1.client %s", err)
		}

		// Fallback to v1 discovery when direct IDs are not provided
		if intTests.testTemplateID == "" {
			var tmpl v1.Template
			v1.FindTemplateForTests(&tmpl, intTests.testPool.ID.String(), "XOA_TEMPLATE")
			intTests.testTemplateID = tmpl.Id
		}
		if intTests.testNetworkID == "" {
			var net v1.Network
			v1.FindNetworkForTests(intTests.testPool.ID.String(), &net)
			intTests.testNetworkID = net.Id
		}
	}

	// Get resource test prefix from environment variable if set
	if prefix, found := os.LookupEnv("XOA_TEST_PREFIX"); found {
		intTestsPrefix = prefix
	}
	// Add time to the test prefix to avoid collisions when running tests in parallel
	intTestsPrefix = fmt.Sprintf("%s%d-", intTestsPrefix, time.Now().Unix())

	slog.Info(fmt.Sprintf("Using test prefix: %s", intTestsPrefix))

	// Run test suite
	code := m.Run()

	// Global teardown
	integrationCancel()

	os.Exit(code)
}

// SetupTestContext prepares the environment for an individual test and returns a context with timeout
func SetupTestContext(t *testing.T) (context.Context, library.Library, string) {
	t.Helper()

	// Create a derived context with timeout for the test
	// #nosec G118 -- cancel() is called in the test cleanup function to ensure it is always called
	ctx, cancel := context.WithTimeout(intTests.ctx, 5*time.Minute)

	// Unique test prefix for this test to avoid to delete resources from other tests
	prefix := intTestsPrefix + t.Name() + "-"

	// Configure logger to use testing.T.Logf via custom sink
	sink := RegisterTestingSink(t)
	testConfig := *intTests.testConfig
	testConfig.LogOutputPaths = []string{sink}
	testConfig.LogErrorOutputPaths = []string{sink}

	// Initialize XO client
	testClient, err := v2.New(&testConfig)
	if err != nil {
		log.Fatalf("test client initialization failed: %v", err)
	}

	// Make the V1Client use t.Logf
	if !intTests.v1Disabled {
		handler := slog.NewTextHandler(&testLogWriter{t}, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		if client, ok := intTests.v1Client.(*v1.Client); ok {
			client.SetLogger(slog.New(handler))
		}
	}

	// Register teardown function
	t.Cleanup(func() {
		cancel() // Cancel the test context
		// Teardown: cleanup any leftover
		_ = cleanupVMsWithPrefix(t, testClient, prefix)
		if !intTests.v1Disabled {
			_ = v1.RemoveNetworksWithNamePrefixForTests(prefix)("")
			_ = v1.RemoveVDIsWithPrefixForTests(prefix)("")
		}
	})

	return ctx, testClient, prefix
}

// findPoolForTests finds a pool by name from the XOA_POOL environment variable
func findPoolForTests() payloads.Pool {
	// Initialize XO client
	client, err := v2.New(intTests.testConfig)
	if err != nil {
		log.Fatalf("test client initialization failed: %v", err)
	}

	poolName, found := os.LookupEnv("XOA_POOL")

	if !found {
		log.Fatal("The XOA_POOL environment variable must be set")
	}

	pools, err := client.Pool().GetAll(intTests.ctx, 0, poolName)
	if err != nil {
		log.Fatalf("failed to get pool with name: %v with error: %v", poolName, err)
	}
	if len(pools) == 0 {
		log.Fatalf("failed to find a pool with name: %v, no poll returned", poolName)
	}
	if len(pools) != 1 {
		log.Fatalf("Found %d pools with name_label %s."+
			"Please use a label that is unique so tests are reproducible.\n", len(pools), poolName)
	}

	return *pools[0]
}

// cleanupVMsWithPrefix removes all VMs that have the testing prefix in their name
func cleanupVMsWithPrefix(t testing.TB, client library.Library, prefix string) error {
	t.Helper()
	vms, err := client.VM().GetAll(intTests.ctx, 0, "name_label:\""+prefix+"\"")
	if err != nil {
		return fmt.Errorf("failed to get VMs: %v", err)
	}

	for _, vm := range vms {
		if vm.NameLabel != "" && vm.ID != uuid.Nil {
			// Check that VM name starts with the test prefix
			if len(vm.NameLabel) >= len(prefix) && (vm.NameLabel)[:len(prefix)] == prefix {
				// t.Logf("Found remaining test VM, Deleting test... NameLabel=%s ID=%s", vm.NameLabel, vm.ID)
				err := client.VM().Delete(intTests.ctx, vm.ID)
				if err != nil {
					t.Logf("failed to delete VM NameLabel=%s error=%v", vm.NameLabel, err)
					return fmt.Errorf("failed to delete VM %s: %v", vm.NameLabel, err)
				}
			}
		}
	}
	return nil
}

// findPBDForTests returns the UUID of the PBD identified by XOA_TEST_PBD_ID, or uuid.Nil if unset.
// Tests that require a specific PBD (e.g. plug/unplug) will be skipped when uuid.Nil.
func findPBDForTests() uuid.UUID {
	raw, ok := os.LookupEnv("XOA_TEST_PBD_ID")
	if !ok || raw == "" {
		return uuid.Nil
	}
	id, err := uuid.FromString(raw)
	if err != nil {
		log.Fatalf("XOA_TEST_PBD_ID=%q is not a valid UUID: %v", raw, err)
	}
	return id
}

// findStorageRepositoryForTests finds a storage repository by name from the XOA_STORAGE environment variable
func findStorageRepositoryForTests() payloads.StorageRepository {
	client, err := v2.New(intTests.testConfig)
	if err != nil {
		log.Fatalf("test client initialization failed: %v", err)
	}

	srName, found := os.LookupEnv("XOA_STORAGE")
	if !found {
		log.Fatalf("XOA_STORAGE environment variable must be set")
	}

	srs, err := client.SR().GetAll(intTests.ctx, 0, "name_label:"+srName+" $pool:"+intTests.testPool.ID.String())
	if err != nil {
		log.Fatalf("failed to get storage repository with name: %s, with err: %v", srName, err)
	}
	if len(srs) == 0 {
		log.Fatalf("failed to find a storage repository with name: %v, no storage repository returned", srName)
	}
	if len(srs) != 1 {
		log.Fatalf("Found %d storage repositories with name_label %s."+
			"Please use a label that is unique so tests are reproducible.\n", len(srs), srName)
	}
	return *srs[0]
}
