package library

import (
	"context"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -destination=mock/task.go -package=mock_library Task,TaskAction

type Task interface {
	Get(ctx context.Context, path string) (*payloads.Task, error)
	List(ctx context.Context, options map[string]any) ([]*payloads.Task, error)

	TaskAction
}

type TaskAction interface {
	Abort(ctx context.Context, id string) error
	Wait(ctx context.Context, id string) (*payloads.Task, error)
	HandleTaskResponse(ctx context.Context, response string) (*payloads.Task, error)
}
