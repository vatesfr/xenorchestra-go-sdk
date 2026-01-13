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

	// TODO: replace v1 struct by payloads.STRUCT when available in v2

	// testTemplate holds a template used for VM creation tests
	testTemplate v1.Template
	// testNetwork holds a network used for network-related tests
	testNetwork v1.Network

	// v1Client is the XO client used for resources not yet available in v2
	// Should not be used to perform the actual test but only to setup/teardown resources
	v1Client v1.XOClient
}

var (
	// intTests holds global test configuration and resources shared across all integration tests
	intTests   integrationTestContext = integrationTestContext{}
	testPrefix string                 = "xo-go-sdk-"
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
	intTests.testConfig, err = config.New()
	if err != nil {
		log.Fatalf("configuration failed: %v", err)
	}

	// Force development mode for tests
	intTests.testConfig.Development = devMode

	// Initialize v1 client for setup/teardown tasks
	intTests.v1Client, err = v1.NewClient(v1.GetConfigFromEnv())
	if err != nil {
		log.Fatalf("error getting v1.client %s", err)
		return
	}

	// Get information for testing
	intTests.testPool = *findPoolForTests()
	v1.FindNetworkForTests(intTests.testPool.ID.String(), &intTests.testNetwork)

	// Replace v1 method with v2 when available
	v1.FindTemplateForTests(&intTests.testTemplate, intTests.testPool.ID.String(), "XOA_TEMPLATE")

	// Get resource test prefix from environment variable if set
	if prefix, found := os.LookupEnv("XOA_TEST_PREFIX"); found {
		testPrefix = prefix
	}
	// Add time to the test prefix to avoid collisions when running tests in parallel
	testPrefix = fmt.Sprintf("%s%d-", testPrefix, time.Now().Unix())

	slog.Info(fmt.Sprintf("Using test prefix: %s", testPrefix))

	// Run test suite
	code := m.Run()

	// Global teardown
	integrationCancel()

	os.Exit(code)
}

// SetupTestContext prepares the environment for an individual test and returns a context with timeout
func SetupTestContext(t *testing.T) (context.Context, func(), library.Library, string) {
	t.Helper()

	// Create a derived context with timeout for the test
	ctx, cancel := context.WithTimeout(intTests.ctx, 5*time.Minute)

	// Unique test prefix for this test to avoid to delete resources from other tests
	prefix := testPrefix + t.Name() + "-"

	// Initialize XO client
	testClient, err := v2.New(intTests.testConfig)
	if err != nil {
		log.Fatalf("test client initialization failed: %v", err)
	}

	// Return the teardown function
	return ctx, func() {
		cancel() // Cancel the test context
		// Teardown: cleanup any leftover test VMs and networks
		_ = cleanupVMsWithPrefix(testClient, prefix)
		_ = v1.RemoveNetworksWithNamePrefixForTests(prefix)
	}, testClient, prefix
}

// findPoolForTests finds a pool by name from the XOA_POOL environment variable
func findPoolForTests() *payloads.Pool {
	// Initialize XO client
	client, err := v2.New(intTests.testConfig)
	if err != nil {
		log.Fatalf("test client initialization failed: %v", err)
	}

	poolName, found := os.LookupEnv("XOA_POOL")

	if !found {
		log.Fatal("The XOA_POOL environment variable must be set")
		os.Exit(-1)
	}

	pools, err := client.Pool().GetAll(intTests.ctx, 0, poolName)
	if err != nil {
		log.Fatalf("failed to get pool with name: %v with error: %v", poolName, err)
		os.Exit(-1)
	}
	if len(pools) == 0 {
		log.Fatalf("failed to find a pool with name: %v, no poll returned", poolName)
		os.Exit(-1)
	}
	if len(pools) != 1 {
		log.Fatalf("Found %d pools with name_label %s."+
			"Please use a label that is unique so tests are reproducible.\n", len(pools), poolName)
		os.Exit(-1)
	}

	return pools[0]
}

// cleanupVMs removes all VMs that have the testing prefix in their name
func cleanupVMsWithPrefix(client library.Library, prefix string) error {
	vms, err := client.VM().GetAll(intTests.ctx, 0, "name_label:"+prefix)
	if err != nil {
		return fmt.Errorf("failed to get VMs: %v", err)
	}

	for _, vm := range vms {
		if vm.NameLabel != "" && vm.ID != uuid.Nil {
			// Check that VM name starts with the test prefix
			if len(vm.NameLabel) >= len(prefix) && (vm.NameLabel)[:len(prefix)] == prefix {
				slog.Info("Found remaining test VM, Deleting test...", "NameLabel", vm.NameLabel, "ID", vm.ID)
				err := client.VM().Delete(intTests.ctx, vm.ID)
				if err != nil {
					slog.Error("failed to delete VM", "NameLabel", vm.NameLabel, "error", err)
					return fmt.Errorf("failed to delete VM %s: %v", vm.NameLabel, err)
				}
			}
		}
	}
	return nil
}
