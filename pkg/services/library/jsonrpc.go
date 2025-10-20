package library

import "go.uber.org/zap"

//go:generate mockgen --build_flags=--mod=mod --destination mock/jsonrpc.go . JSONRPC
type JSONRPC interface {
	Call(method string, params map[string]any, result any, logContext ...zap.Field) error
	ValidateResult(result bool, operation string, logContext ...zap.Field) error
}
