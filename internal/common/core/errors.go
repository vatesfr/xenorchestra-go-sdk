package core

import "fmt"

// ClientError is a type for errors that occur in the client package.
// It is a string that can be formatted with arguments. It avoids to
// repeat the error message formatted in the client code.
type ClientError string

const (
	ErrFailedToUnmarshalResponse ClientError = "failed to unmarshal response %s"
	ErrFailedToMarshalResponse   ClientError = "failed to marshal response %s"
	ErrFailedToReadResponse      ClientError = "failed to read response body %s"

	ErrFailedToUnmarshalParams ClientError = "failed to unmarshal params %s"
	ErrFailedToMarshalParams   ClientError = "failed to marshal params %s"

	ErrFailedToMakeRequest ClientError = "failed to make request %s"

	ErrFailedToParseURL         ClientError = "failed to parse URL %s"
	ErrFailedToSetHeader        ClientError = "failed to set header %s"
	ErrFailedToDoRequest        ClientError = "failed to do request %s"
	ErrFailedToReadResponseBody ClientError = "failed to read response body %s"

	ErrUnexpectedResponseType ClientError = "unexpected response type %T"
)

// WithArgs returns a new error with the given arguments.
func (e ClientError) WithArgs(args ...any) error {
	return fmt.Errorf(string(e), args...)
}
