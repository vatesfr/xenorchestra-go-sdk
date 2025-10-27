package jsonrpc

import (
	"fmt"

	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"go.uber.org/zap"
)

type Service struct {
	client *v1.Client
	log    *logger.Logger
}

func New(client *v1.Client, log *logger.Logger) library.JSONRPC {
	return &Service{
		client: client,
		log:    log,
	}
}

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
