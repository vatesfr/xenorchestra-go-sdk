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

	// testTemplate holds the template used for VM creation tests.
	// Resolved from XOA_TEMPLATE_ID (direct) or XOA_TEMPLATE name lookup (fallback).
	testTemplate payloads.Template

	// testNetworkID holds the network UUID used for network-related tests.
	// Resolved from XOA_NETWORK_ID (direct) or XOA_NETWORK name lookup (fallback).
	testNetworkID uuid.UUID

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
	// - XOA_DISABLE_V1: true to disable v1 client
	// - XOA_TEMPLATE_ID: direct template UUID (takes precedence over XOA_TEMPLATE name lookup)
	// - XOA_NETWORK_ID: direct network UUID (takes precedence over XOA_NETWORK name lookup)
	intTests.testConfig, err = config.New()
	if err != nil {
		log.Fatalf("configuration failed: %v", err)
	}

	// Force development mode for tests
	intTests.testConfig.Development = devMode

	// Determine v1 status
	intTests.v1Disabled, _ = strconv.ParseBool(os.Getenv("XOA_DISABLE_V1"))

	// Resolve network ID: direct env var takes precedence, fallback to XOA_NETWORK name lookup
	if networkID, found := os.LookupEnv("XOA_NETWORK_ID"); found && networkID != "" {
		intTests.testNetworkID = uuid.Must(uuid.FromString(networkID))
	}

	// Get information for testing
	setupClient, err := v2.New(intTests.testConfig)
	if err != nil {
		log.Fatalf("test client initialization failed: %v", err)
	}
	pool := findUniqueByLabel(setupClient, "pool", "XOA_POOL", func() ([]*payloads.Pool, error) {
		return setupClient.Pool().GetAll(intTests.ctx, 0, os.Getenv("XOA_POOL"))
	})
	intTests.testPool = *pool
	srName := os.Getenv("XOA_STORAGE")
	getSR := func() ([]*payloads.StorageRepository, error) {
		filter := "name_label:\"" + srName + "\" $pool:" + intTests.testPool.ID.String()
		return setupClient.SR().GetAll(intTests.ctx, 0, filter)
	}
	intTests.testSR = *findUniqueByLabel(setupClient, "storage repository", "XOA_STORAGE", getSR)
	intTests.testPBD = findPBDForTests()
	intTests.testTemplate = findTemplateForTests(setupClient)
	if intTests.testNetworkID == uuid.Nil {
		intTests.testNetworkID = findNetworkForTests(setupClient).ID
	}

	// Initialize v1 client only when needed for v1-dependent tests
	if !intTests.v1Disabled {
		intTests.v1Client, err = v1.NewClientWithLogger(v1.GetConfigFromEnv(), logger)
		if err != nil {
			integrationCancel()
			log.Fatalf("error getting v1.client %s", err)
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
		_ = cleanupNetworksWithPrefix(t, testClient, prefix)
		_ = cleanupVDIWithPrefix(t, testClient, prefix)
	})

	return ctx, testClient, prefix
}

// findUniqueByLabel looks up exactly one resource of the given kind by
// name_label in the test pool, using the value of the given environment
// variable. It fails the suite if the variable is unset or the match is not
// unique.
func findUniqueByLabel[T any](client library.Library, kind, envVar string, get func() ([]T, error)) T {
	label, found := os.LookupEnv(envVar)
	if !found {
		log.Fatalf("%s environment variable must be set", envVar)
	}

	items, err := get()
	if err != nil {
		log.Fatalf("failed to get %s with name: %s, with err: %v", kind, label, err)
	}
	if len(items) == 0 {
		log.Fatalf("failed to find a %s with name: %v, no %s returned", kind, label, kind)
	}
	if len(items) != 1 {
		log.Fatalf("Found %d %ss with name_label %s."+
			"Please use a label that is unique so tests are reproducible.\n", len(items), kind, label)
	}
	return items[0]
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

func cleanupNetworksWithPrefix(t testing.TB, client library.Library, prefix string) error {
	t.Helper()
	networks, err := client.Network().GetAll(intTests.ctx, 0, "name_label:\""+prefix+"\"")
	if err != nil {
		return fmt.Errorf("failed to get networks: %v", err)
	}

	for _, network := range networks {
		// Check that network name starts with the test prefix
		if len(network.NameLabel) >= len(prefix) && (network.NameLabel)[:len(prefix)] == prefix {
			// t.Logf("Found remaining test Network, Deleting test... NameLabel=%s ID=%s", network.NameLabel, network.ID)
			err := client.Network().Delete(intTests.ctx, network.ID)
			if err != nil {
				t.Logf("failed to delete Network NameLabel=%s error=%v", network.NameLabel, err)
				return fmt.Errorf("failed to delete Network %s: %v", network.NameLabel, err)
			}
		}
	}
	return nil
}

func cleanupVDIWithPrefix(t testing.TB, client library.Library, prefix string) error {
	t.Helper()
	vdis, err := client.VDI().GetAll(intTests.ctx, 0, "name_label:\""+prefix+"\"")
	if err != nil {
		return fmt.Errorf("failed to get VDIs: %v", err)
	}

	for _, vdi := range vdis {
		// Check that VDI name starts with the test prefix
		if len(vdi.NameLabel) >= len(prefix) && (vdi.NameLabel)[:len(prefix)] == prefix {
			// t.Logf("Found remaining test VDI, Deleting test... NameLabel=%s ID=%s", vdi.NameLabel, vdi.ID)
			err := client.VDI().Delete(intTests.ctx, vdi.ID)
			if err != nil {
				t.Logf("failed to delete VDI NameLabel=%s error=%v", vdi.NameLabel, err)
				return fmt.Errorf("failed to delete VDI %s: %v", vdi.NameLabel, err)
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

// findTemplateForTests resolves the test template: XOA_TEMPLATE_ID (direct
// UUID) takes precedence, falling back to an XOA_TEMPLATE name lookup.
func findTemplateForTests(client library.Library) payloads.Template {
	if templateID, found := os.LookupEnv("XOA_TEMPLATE_ID"); found && templateID != "" {
		tmpl, err := client.Template().Get(intTests.ctx, templateID)
		if err != nil {
			log.Fatalf("failed to get template with ID: %s, with err: %v", templateID, err)
		}
		return *tmpl
	}

	tplName := os.Getenv("XOA_TEMPLATE")
	tpl := findUniqueByLabel(client, "template", "XOA_TEMPLATE", func() ([]*payloads.Template, error) {
		filter := "name_label:\"" + tplName + "\" $pool:" + intTests.testPool.ID.String()
		return client.Template().GetAll(intTests.ctx, 0, filter)
	})
	return *tpl
}

func findNetworkForTests(client library.Library) payloads.Network {
	netName := os.Getenv("XOA_NETWORK")
	net := findUniqueByLabel(client, "network", "XOA_NETWORK", func() ([]*payloads.Network, error) {
		filter := "name_label:\"" + netName + "\" $pool:" + intTests.testPool.ID.String()
		return client.Network().GetAll(intTests.ctx, 0, filter)
	})
	return *net
}
