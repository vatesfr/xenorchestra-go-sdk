package integration

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	v2 "github.com/vatesfr/xenorchestra-go-sdk/v2"
)

// Global client instance that will be reused across all tests
var (
	globalClient      library.Library
	clientMutex       sync.Mutex
	clientInitialized bool
)

const (
	// TODO: this should be a boolean
	trueStr = "true"
)

// initializeClient initializes the global client instance if it hasn't been initialized yet
func initializeClient() (library.Library, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if clientInitialized && globalClient != nil {
		return globalClient, nil
	}

	cfg, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	client, err := v2.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create XO client: %w", err)
	}

	globalClient = client
	clientInitialized = true

	return client, nil
}

type TestClient struct {
	Client       library.Library
	Pool         string
	Template     string
	Network      string
	Storage      string
	PoolID       string
	TemplateID   string
	NetworkID    string
	StorageID    string
	TestPrefix   string
	SkipTeardown bool
}

func Setup(t *testing.T) *TestClient {
	if os.Getenv("XOA_INTEGRATION_TESTS") != trueStr {
		t.Skip("Skipping integration test. Set XOA_INTEGRATION_TESTS=" + trueStr + " to run")
	}

	client, err := initializeClient()
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	testPrefix := os.Getenv("XOA_TEST_PREFIX")
	if testPrefix == "" {
		testPrefix = "go-sdk-test"
	}

	tc := &TestClient{
		Client:       client,
		Pool:         os.Getenv("XOA_POOL"),
		Template:     os.Getenv("XOA_TEMPLATE"),
		Network:      os.Getenv("XOA_NETWORK"),
		Storage:      os.Getenv("XOA_STORAGE"),
		PoolID:       os.Getenv("XOA_POOL_ID"),
		TemplateID:   os.Getenv("XOA_TEMPLATE_ID"),
		NetworkID:    os.Getenv("XOA_NETWORK_ID"),
		StorageID:    os.Getenv("XOA_STORAGE_ID"),
		TestPrefix:   testPrefix,
		SkipTeardown: os.Getenv("XOA_SKIP_TEARDOWN") == trueStr,
	}

	tc.validateEnvironment(t)

	return tc
}

func (tc *TestClient) validateEnvironment(t *testing.T) {
	missingVars := []string{}

	if tc.Pool == "" && tc.PoolID == "" {
		missingVars = append(missingVars, "XOA_POOL or XOA_POOL_ID")
	}

	if tc.Template == "" && tc.TemplateID == "" {
		missingVars = append(missingVars, "XOA_TEMPLATE or XOA_TEMPLATE_ID")
	}

	if tc.Network == "" && tc.NetworkID == "" {
		missingVars = append(missingVars, "XOA_NETWORK or XOA_NETWORK_ID")
	}

	if tc.Storage == "" && tc.StorageID == "" {
		t.Log("Neither XOA_STORAGE nor XOA_STORAGE_ID is set. Backup tests may be skipped.")
	}

	if len(missingVars) > 0 {
		t.Fatalf("Missing required environment variables: %v", missingVars)
	}
}

func (tc *TestClient) GenerateResourceName(resourceType string) string {
	return fmt.Sprintf("%s-%s-%d", tc.TestPrefix, resourceType, os.Getpid())
}

func GetUUID(t *testing.T, id string) uuid.UUID {
	uid, err := uuid.FromString(id)
	if err != nil {
		t.Fatalf("Invalid UUID %s: %v", id, err)
	}
	return uid
}

func (tc *TestClient) CleanupVM(t *testing.T, nameLabel string) {
	if tc.SkipTeardown {
		t.Logf("Skipping cleanup of VM %s", nameLabel)
		return
	}

	ctx := context.Background()
	vms, err := tc.Client.VM().List(ctx, map[string]any{"limit": 10})
	if err != nil {
		t.Logf("Failed to list VMs for cleanup: %v", err)
		return
	}

	for _, vm := range vms {
		if vm.NameLabel == nameLabel {
			err := tc.Client.VM().Delete(ctx, vm.ID)
			if err != nil {
				t.Logf("Failed to delete VM %s: %v", nameLabel, err)
			} else {
				t.Logf("Successfully deleted VM %s", nameLabel)
			}
			return
		}
	}

	t.Logf("VM %s not found for cleanup", nameLabel)
}
