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

var (
	// integrationCtx is the parent context for all tests
	integrationCtx    context.Context
	integrationCancel context.CancelFunc
	testClient        library.Library
	testPool          payloads.Pool

	// TODO: replace v1 struct by payloads.STRUCT when available in v2
	testTemplate v1.Template
	testNetwork  v1.Network
	v1TestClient v1.XOClient // Used for resources not yet available in v2

	integrationTestPrefix string = "xo-go-sdk-"
)

// TestMain is the main entry point for integration tests
func TestMain(m *testing.M) {
	var err error
	// Global setup
	integrationCtx, integrationCancel = context.WithCancel(context.Background())

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
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("configuration failed: %v", err)
	}

	// Force development mode for tests
	cfg.Development = devMode

	// Initialize XO client
	testClient, err = v2.New(cfg)
	if err != nil {
		log.Fatalf("test client initialization failed: %v", err)
	}
	v1TestClient, err = v1.NewClient(v1.GetConfigFromEnv())
	if err != nil {
		log.Fatalf("error deleting network: error getting v1.client %s", err)
		return
	}

	// Get information for testing
	findPoolForTests(&testPool)
	v1.FindNetworkForTests(testPool.ID.String(), &testNetwork)

	// Replace v1 method with v2 when available
	v1.FindTemplateForTests(&testTemplate, testPool.ID.String(), "XOA_TEMPLATE")

	// Get resource test prefix from environment variable if set
	if prefix, found := os.LookupEnv("XOA_TEST_PREFIX"); found {
		integrationTestPrefix = prefix
	}
	// Add time to the test prefix to avoid collisions when running tests in parallel
	integrationTestPrefix = fmt.Sprintf("%s%d-", integrationTestPrefix, time.Now().Unix())

	slog.Info(fmt.Sprintf("Using test prefix: %s", integrationTestPrefix))

	// Run test suite
	code := m.Run()

	// Global teardown
	integrationCancel()

	os.Exit(code)
}

// SetupTestContext prepares the environment for an individual test and returns a context with timeout
func SetupTestContext(t *testing.T) (context.Context, func()) {
	t.Helper()

	// Create a derived context with timeout for the test
	ctx, cancel := context.WithTimeout(integrationCtx, 5*time.Minute)

	// Return the teardown function
	return ctx, func() {
		cancel() // Cancel the test context
		// Teardown: cleanup any leftover test VMs and networks
		_ = cleanupVMsWithPrefix(integrationTestPrefix)
		_ = v1.RemoveNetworksWithNamePrefixForTests(integrationTestPrefix)
	}
}

// findPoolForTests finds a pool by name from the XOA_POOL environment variable
func findPoolForTests(pool *payloads.Pool) {
	poolName, found := os.LookupEnv("XOA_POOL")

	if !found {
		log.Fatal("The XOA_POOL environment variable must be set")
		os.Exit(-1)
	}

	pools, err := testClient.Pool().GetAll(integrationCtx, 0, poolName)
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

	*pool = *pools[0]
}

// CleanupVMs removes all VMs that have the testing prefix in their name
func cleanupVMsWithPrefix(prefix string) error {
	vms, err := testClient.VM().List(integrationCtx, 0, "name_label:"+prefix)
	if err != nil {
		return fmt.Errorf("failed to get VMs: %v", err)
	}

	for _, vm := range vms {
		if vm.NameLabel != "" && vm.ID != uuid.Nil {
			// Check that VM name starts with the test prefix
			if len(vm.NameLabel) >= len(prefix) && (vm.NameLabel)[:len(prefix)] == prefix {
				slog.Info("Found remaining test VM, Deleting test...", "NameLabel", vm.NameLabel, "ID", vm.ID)
				err := testClient.VM().Delete(integrationCtx, vm.ID)
				if err != nil {
					slog.Error("failed to delete VM", "NameLabel", vm.NameLabel, "error", err)
					return fmt.Errorf("failed to delete VM %s: %v", vm.NameLabel, err)
				}
			}
		}
	}
	return nil
}
