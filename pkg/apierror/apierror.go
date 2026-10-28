// Package apierror defines the error returned by the v2 client when the Xen
// Orchestra REST API answers with a non-2xx status.
//
// Callers inspect it with errors.As (or the As helper) instead of parsing the
// error text:
//
//	vm, err := client.VM().GetByID(ctx, id)
//	if apierror.IsNoSuchObject(err, payloads.ResourceTypeVM, id.String()) {
//		// Xen Orchestra itself says this VM does not exist.
//	}
package apierror

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

// Error is a non-2xx answer of the Xen Orchestra REST API.
type Error struct {
	// StatusCode is the HTTP status code, e.g. 404.
	StatusCode int
	// Status is the HTTP status line, e.g. "404 Not Found".
	Status string
	// Method is the HTTP method of the request, e.g. "GET".
	Method string
	// URL is the request URL without its query string.
	URL string
	// Body is the raw response body.
	Body []byte
	// Message is the "error" field of a Xen Orchestra JSON error body.
	// It is empty when the body is not a Xen Orchestra error (for example
	// an HTML page answered by a reverse proxy).
	Message string
	// Data is the "data" field of a Xen Orchestra JSON error body, nil when
	// absent.
	Data json.RawMessage
}

// xoErrorBody is the JSON body written by the Xen Orchestra REST API error
// handler (@xen-orchestra/rest-api generic-error-handler.middleware).
type xoErrorBody struct {
	Error string          `json:"error"`
	Data  json.RawMessage `json:"data"`
}

// NewFromResponse builds an Error from the parts of an HTTP answer. The body
// is decoded leniently: a body that is not a Xen Orchestra JSON error leaves
// Message and Data empty.
func NewFromResponse(method, rawURL string, statusCode int, status string, body []byte) *Error {
	if status == "" {
		status = fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode))
	}

	e := &Error{
		StatusCode: statusCode,
		Status:     status,
		Method:     method,
		URL:        stripQuery(rawURL),
		Body:       body,
	}

	var xo xoErrorBody
	if err := json.Unmarshal(body, &xo); err == nil {
		e.Message = xo.Error
		if len(xo.Data) > 0 && string(xo.Data) != "null" {
			e.Data = xo.Data
		}
	}

	return e
}

// Error returns "API error: <status> - <body>", the text the client has
// always used, so callers that still match on it keep working.
func (e *Error) Error() string {
	if e == nil {
		return "API error: <nil>"
	}

	return fmt.Sprintf("API error: %s - %s", e.Status, string(e.Body))
}

// NoSuchObject is the data of Xen Orchestra's noSuchObject error
// (xo-common api-errors: `no such ${type || 'object'} ${id}`).
type NoSuchObject struct {
	// ID is the id of the object that was looked up.
	ID string `json:"id"`
	// Type is the Xen Orchestra object type (payloads.ResourceTypeVM...).
	// Xen Orchestra omits it when the lookup was not typed.
	Type payloads.ResourceType `json:"type,omitempty"`
}

// NoSuchObject returns the object Xen Orchestra reports as missing. It is
// false unless the status is 404 and the body carries data.id.
func (e *Error) NoSuchObject() (NoSuchObject, bool) {
	if e == nil || e.StatusCode != http.StatusNotFound || len(e.Data) == 0 {
		return NoSuchObject{}, false
	}

	var nso NoSuchObject
	if err := json.Unmarshal(e.Data, &nso); err != nil || nso.ID == "" {
		return NoSuchObject{}, false
	}

	return nso, true
}

// As returns the *Error in err's chain, if any. It never returns a nil
// *Error with ok=true: a typed-nil *Error stored in a non-nil error interface
// would otherwise match and let callers dereference nil.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) && e != nil {
		return e, true
	}

	return nil, false
}

// IsNotFound reports whether err is a 404 answer, whatever its body. A 404
// may come from something in front of Xen Orchestra (a reverse proxy, a wrong
// URL); use IsNoSuchObject to know that Xen Orchestra itself does not have an
// object.
func IsNotFound(err error) bool {
	e, ok := As(err)

	return ok && e.StatusCode == http.StatusNotFound
}

// IsNoSuchObject reports whether err is Xen Orchestra's own 404 for the
// object id: the body's data.id equals id (case-insensitive) and, when Xen
// Orchestra sent a type, data.type equals objType.
// Nothing but Xen Orchestra can echo the requested id, so a 404 from another
// server is never mistaken for a missing object.
func IsNoSuchObject(err error, objType payloads.ResourceType, id string) bool {
	e, ok := As(err)
	if !ok {
		return false
	}

	nso, ok := e.NoSuchObject()
	if !ok || !strings.EqualFold(nso.ID, id) {
		return false
	}

	return nso.Type == "" || nso.Type == objType
}

func stripQuery(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		if i := strings.IndexByte(rawURL, '?'); i >= 0 {
			return rawURL[:i]
		}

		return rawURL
	}

	u.RawQuery = ""
	u.Fragment = ""

	return u.String()
}
