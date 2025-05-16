package core

// EmptyParams is an empty struct that can be used
// to represent no parameters instead of passing
// an empty struct like struct{}{}. We can then
// check using reflect if the parameter is empty
var EmptyParams struct{}

var EmptyResult struct{}

// JsonRpcPayload is a struct that represents a JSON-RPC payload.
// It is used to convert a Go struct or map to a JSON-RPC compatible map.
type JsonRpcPayload struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// JsonRpcError is a struct that represents a JSON-RPC error.
// It is used to convert a Go struct or map to a JSON-RPC compatible map.
type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
