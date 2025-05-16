package schedule

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	mock_library "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library/mock"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func setupScheduleTest(t *testing.T) (library.Schedule, *mock_library.MockJSONRPC) {
	ctrl := gomock.NewController(t)
	mockJSONRPC := mock_library.NewMockJSONRPC(ctrl)
	log, _ := logger.New(false)
	scheduleService := New(mockJSONRPC, log)
	return scheduleService, mockJSONRPC
}

func TestGet(t *testing.T) {
	service, mockJSONRPC := setupScheduleTest(t)
	ctx := context.Background()

	t.Run("existing schedule", func(t *testing.T) {
		scheduleID := uuid.Must(uuid.NewV4())
		expectedSchedule := &payloads.Schedule{
			ID:       scheduleID,
			Name:     "test-schedule",
			Cron:     "0 0 * * *",
			Enabled:  true,
			Timezone: "UTC",
			JobID:    uuid.Must(uuid.NewV4()),
		}

		mockJSONRPC.EXPECT().
			Call("schedule.get", map[string]any{"id": scheduleID}, gomock.Any(), gomock.Any()).
			DoAndReturn(func(method string, params map[string]any, result any, logContext ...zap.Field) error {
				*(result.(*payloads.Schedule)) = *expectedSchedule
				return nil
			})

		schedule, err := service.Get(ctx, scheduleID)
		assert.NoError(t, err)
		assert.Equal(t, expectedSchedule, schedule)
	})

	t.Run("nonexistent schedule", func(t *testing.T) {
		scheduleID := uuid.Must(uuid.NewV4())
		mockJSONRPC.EXPECT().
			Call("schedule.get", map[string]any{"id": scheduleID}, gomock.Any(), gomock.Any()).
			Return(errors.New("schedule not found"))

		schedule, err := service.Get(ctx, scheduleID)
		assert.Error(t, err)
		assert.Nil(t, schedule)
	})
}

func TestGetAll(t *testing.T) {
	service, mockJSONRPC := setupScheduleTest(t)
	ctx := context.Background()

	expectedSchedules := []*payloads.Schedule{
		{
			ID:       uuid.Must(uuid.NewV4()),
			Name:     "schedule-1",
			Cron:     "0 0 * * *",
			Enabled:  true,
			Timezone: "UTC",
			JobID:    uuid.Must(uuid.NewV4()),
		},
		{
			ID:       uuid.Must(uuid.NewV4()),
			Name:     "schedule-2",
			Cron:     "0 12 * * *",
			Enabled:  false,
			Timezone: "UTC",
			JobID:    uuid.Must(uuid.NewV4()),
		},
	}

	mockJSONRPC.EXPECT().
		Call("schedule.getAll", map[string]any{}, gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result any, logContext ...zap.Field) error {
			*(result.(*[]*payloads.Schedule)) = expectedSchedules
			return nil
		})

	schedules, err := service.GetAll(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedSchedules, schedules)
}

func TestCreate(t *testing.T) {
	service, mockJSONRPC := setupScheduleTest(t)
	ctx := context.Background()

	newSchedule := &payloads.Schedule{
		Name:     "new-schedule",
		Cron:     "0 0 * * *",
		Enabled:  true,
		Timezone: "UTC",
		JobID:    uuid.Must(uuid.NewV4()),
	}

	expectedSchedule := &payloads.Schedule{
		ID:       uuid.Must(uuid.NewV4()),
		Name:     newSchedule.Name,
		Cron:     newSchedule.Cron,
		Enabled:  newSchedule.Enabled,
		Timezone: newSchedule.Timezone,
		JobID:    newSchedule.JobID,
	}

	mockJSONRPC.EXPECT().
		Call("schedule.create", map[string]any{
			"name":     newSchedule.Name,
			"cron":     newSchedule.Cron,
			"enabled":  newSchedule.Enabled,
			"timezone": newSchedule.Timezone,
			"jobId":    newSchedule.JobID,
		}, gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result any, logContext ...zap.Field) error {
			*(result.(*payloads.Schedule)) = *expectedSchedule
			return nil
		})

	createdSchedule, err := service.Create(ctx, newSchedule)
	assert.NoError(t, err)
	assert.Equal(t, expectedSchedule, createdSchedule)
}

func TestUpdate(t *testing.T) {
	service, mockJSONRPC := setupScheduleTest(t)
	ctx := context.Background()

	scheduleID := uuid.Must(uuid.NewV4())
	updatedSchedule := &payloads.Schedule{
		Name:     "updated-schedule",
		Cron:     "0 12 * * *",
		Enabled:  false,
		Timezone: "UTC",
		JobID:    uuid.Must(uuid.NewV4()),
	}

	expectedSchedule := &payloads.Schedule{
		ID:       scheduleID,
		Name:     updatedSchedule.Name,
		Cron:     updatedSchedule.Cron,
		Enabled:  updatedSchedule.Enabled,
		Timezone: updatedSchedule.Timezone,
		JobID:    updatedSchedule.JobID,
	}

	// Mock the update call
	mockJSONRPC.EXPECT().
		Call("schedule.set", map[string]any{
			"id":       scheduleID,
			"name":     updatedSchedule.Name,
			"cron":     updatedSchedule.Cron,
			"enabled":  updatedSchedule.Enabled,
			"timezone": updatedSchedule.Timezone,
			"jobId":    updatedSchedule.JobID,
		}, gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result any, logContext ...zap.Field) error {
			*(result.(*bool)) = true
			return nil
		})

	// Mock the get call to return the updated schedule
	mockJSONRPC.EXPECT().
		Call("schedule.get", map[string]any{"id": scheduleID}, gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result any, logContext ...zap.Field) error {
			*(result.(*payloads.Schedule)) = *expectedSchedule
			return nil
		})

	updated, err := service.Update(ctx, scheduleID, updatedSchedule)
	assert.NoError(t, err)
	assert.Equal(t, expectedSchedule, updated)
}

func TestDelete(t *testing.T) {
	service, mockJSONRPC := setupScheduleTest(t)
	ctx := context.Background()

	scheduleID := uuid.Must(uuid.NewV4())

	mockJSONRPC.EXPECT().
		Call("schedule.delete", map[string]any{"id": scheduleID}, nil, gomock.Any()).
		Return(nil)

	err := service.Delete(ctx, scheduleID)
	assert.NoError(t, err)
}
