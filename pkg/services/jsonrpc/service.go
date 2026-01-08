package jsonrpc

import (
	"fmt"
	"sync"

	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"go.uber.org/zap"
)

// Service wraps a v1 client and provides JSON-RPC functionality.
type Service struct {
	client *v1.Client
	log    *logger.Logger
}

// LazyService defers v1 client initialization until the first Call().
type LazyService struct {
	Service
	factoryOnce sync.Once
	factory     func() (*v1.Client, error)
	initErr     error
}

// New creates a JSONRPC service with an already-initialized v1 client.
func New(client *v1.Client, log *logger.Logger) library.JSONRPC {
	return &Service{
		client: client,
		log:    log,
	}
}

// NewLazy creates a JSONRPC service that initializes the v1 client lazily
// on first Call(). The factory function will be called at most once, thread-safely.
func NewLazy(factory func() (*v1.Client, error), log *logger.Logger) library.JSONRPC {
	return &LazyService{
		Service: Service{
			log: log,
		},
		factory: factory,
	}
}

// Call performs the actual JSON-RPC call with logging.
func (s *Service) Call(method string, params map[string]any, result any, logContext ...zap.Field) error {
	s.log.Debug("Making JSON-RPC call",
		append([]zap.Field{
			zap.String("method", method),
			zap.Any("params", params),
		}, logContext...)...)

	err := s.client.Call(method, params, result)
	if err != nil {
		s.log.Error("JSON-RPC call failed",
			append([]zap.Field{
				zap.String("method", method),
				zap.Error(err),
			}, logContext...)...)
		return fmt.Errorf("JSON-RPC call to %s failed: %w", method, err)
	}

	s.log.Debug("JSON-RPC call successful",
		append([]zap.Field{
			zap.String("method", method),
			zap.Any("result", result),
		}, logContext...)...)

	return nil
}

// ValidateResult validates a boolean result from a JSON-RPC call.
func (s *Service) ValidateResult(result bool, operation string, logContext ...zap.Field) error {
	if !result {
		s.log.Warn("Operation returned unsuccessful status",
			append([]zap.Field{
				zap.String("operation", operation),
			}, logContext...)...)
		return fmt.Errorf("%s returned unsuccessful status", operation)
	}
	return nil
}

// Call implements library.JSONRPC interface for LazyService.
// It lazily initializes the v1 client on first call.
func (s *LazyService) Call(method string, params map[string]any, result any, logContext ...zap.Field) error {
	// Initialize client lazily on first call
	s.factoryOnce.Do(func() {
		s.Service.client, s.initErr = s.factory()
	})

	if s.initErr != nil {
		s.log.Error("Failed to initialize v1 client",
			append([]zap.Field{
				zap.String("method", method),
				zap.Error(s.initErr),
			}, logContext...)...)
		return fmt.Errorf("failed to initialize v1 client for JSON-RPC call to %s: %w", method, s.initErr)
	}

	return s.Call(method, params, result, logContext...)
}
