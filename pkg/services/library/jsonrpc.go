package library

import "go.uber.org/zap"

//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -destination=mock/jsonrpc.go -package=mock_library JSONRPC
type JSONRPC interface {
	Call(method string, params map[string]any, result any, logContext ...zap.Field) error
	ValidateResult(result bool, operation string, logContext ...zap.Field) error
}
