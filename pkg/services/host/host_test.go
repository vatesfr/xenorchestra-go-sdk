package host

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

// This is a basic test to ensure the service can be instantiated.
// More comprehensive tests would require mocking the client or a real environment.
func TestNew(t *testing.T) {
	cfg := &config.Config{
		Url:   "http://localhost",
		Token: "test-token",
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)

	log, _ := logger.New(true, nil, nil)
	svc := New(c, log)

	assert.NotNil(t, svc)
}

func TestHostService_Get_InvalidUUID(t *testing.T) {
	// This test mainly checks that the code compiles and imports are correct
	// Ideally we would mock the client to test the logic
	cfg := &config.Config{
		Url:   "http://localhost",
		Token: "test-token",
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)

	log, _ := logger.New(true, nil, nil)
	svc := New(c, log)

	_, err = svc.Get(context.Background(), uuid.Nil)
	// Since we don't have a real server, we expect an error or it to try to connect
	// In this unit test setup without mocks, we mainly ensure type safety
	assert.Error(t, err)
}
