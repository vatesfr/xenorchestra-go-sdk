package library

import (
	"context"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/task.go . Task,TaskAction

type Task interface {
	Get(ctx context.Context, path string) (*payloads.Task, error)
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Task, error)

	TaskAction
}

type TaskAction interface {
	Abort(ctx context.Context, id string) error
	Wait(ctx context.Context, id string) (*payloads.Task, error)
	HandleTaskResponse(ctx context.Context, response string, waitForCompletion bool) (*payloads.Task, bool, error)
}
