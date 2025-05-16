package schedule

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"go.uber.org/zap"
)

type Service struct {
	jsonrpcSrv library.JSONRPC
	log        *logger.Logger
}

func New(
	jsonrpcSrv library.JSONRPC,
	log *logger.Logger,
) library.Schedule {
	return &Service{
		jsonrpcSrv: jsonrpcSrv,
		log:        log,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*payloads.Schedule, error) {
	var result payloads.Schedule
	if err := s.jsonrpcSrv.Call("schedule.get", map[string]any{"id": id}, &result, zap.String("scheduleID", id.String())); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) GetAll(ctx context.Context) ([]*payloads.Schedule, error) {
	var result []*payloads.Schedule

	params := map[string]any{}

	if err := s.jsonrpcSrv.Call("schedule.getAll", params, &result); err != nil {
		s.log.Error("Failed to get schedules", zap.Error(err))
		return nil, err
	}

	s.log.Debug("Retrieved schedules", zap.Int("count", len(result)))
	return result, nil
}

func (s *Service) Create(ctx context.Context, schedule *payloads.Schedule) (*payloads.Schedule, error) {
	var result payloads.Schedule
	if err := s.jsonrpcSrv.Call("schedule.create", map[string]any{
		"name":     schedule.Name,
		"cron":     schedule.Cron,
		"enabled":  schedule.Enabled,
		"timezone": schedule.Timezone,
		"jobId":    schedule.JobID,
	}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, schedule *payloads.Schedule) (*payloads.Schedule, error) {
	var success bool
	if err := s.jsonrpcSrv.Call("schedule.set", map[string]any{
		"id":       id,
		"name":     schedule.Name,
		"cron":     schedule.Cron,
		"enabled":  schedule.Enabled,
		"timezone": schedule.Timezone,
		"jobId":    schedule.JobID,
	}, &success); err != nil {
		return nil, err
	}

	if !success {
		return nil, fmt.Errorf("failed to update schedule")
	}

	// Get the updated schedule
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.jsonrpcSrv.Call("schedule.delete",
		map[string]any{"id": id}, nil, zap.String("scheduleID", id.String())); err != nil {
		return err
	}
	return nil
}
