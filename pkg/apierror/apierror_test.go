package apierror

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

const (
	typeVM     = string(payloads.ResourceTypeVM)
	testVMID   = "48b8425b-469a-4b4b-860e-635568e5445a"
	otherVMID  = "550e8400-e29b-41d4-a716-446655440001"
	testVMURL  = "https://xo.example/rest/v0/vms/" + testVMID
	notFound   = "404 Not Found"
	htmlBody   = "<html>default backend - 404</html>"
	methodGET  = http.MethodGet
	statusText = "500 Internal Server Error"
)

func xoNoSuchObjectBody(id, typ string) []byte {
	if typ == "" {
		return []byte(`{"error":"no such object ` + id + `","data":{"id":"` + id + `"}}`)
	}

	return []byte(`{"error":"no such ` + typ + " " + id + `","data":{"id":"` + id + `","type":"` + typ + `"}}`)
}

func TestNewFromResponse(t *testing.T) {
	t.Run("XO noSuchObject body", func(t *testing.T) {
		body := xoNoSuchObjectBody(testVMID, typeVM)
		e := NewFromResponse(methodGET, testVMURL+"?fields=*", http.StatusNotFound, notFound, body)

		assert.Equal(t, http.StatusNotFound, e.StatusCode)
		assert.Equal(t, notFound, e.Status)
		assert.Equal(t, methodGET, e.Method)
		assert.Equal(t, testVMURL, e.URL, "query string is stripped")
		assert.Equal(t, "no such VM "+testVMID, e.Message)
		assert.JSONEq(t, `{"id":"`+testVMID+`","type":"VM"}`, string(e.Data))
	})

	t.Run("XO error without data", func(t *testing.T) {
		body := []byte(`{"error":"invalid parameters"}`)
		e := NewFromResponse(methodGET, testVMURL, http.StatusBadRequest, "400 Bad Request", body)

		assert.Equal(t, "invalid parameters", e.Message)
		assert.Nil(t, e.Data)
	})

	t.Run("XO error with null data", func(t *testing.T) {
		e := NewFromResponse(methodGET, testVMURL, http.StatusNotFound, notFound, []byte(`{"error":"x","data":null}`))

		assert.Nil(t, e.Data)
	})

	t.Run("HTML body", func(t *testing.T) {
		e := NewFromResponse(methodGET, testVMURL, http.StatusNotFound, notFound, []byte(htmlBody))

		assert.Empty(t, e.Message)
		assert.Nil(t, e.Data)
		assert.Equal(t, []byte(htmlBody), e.Body)
	})

	t.Run("empty body and status", func(t *testing.T) {
		e := NewFromResponse(methodGET, testVMURL, http.StatusServiceUnavailable, "", nil)

		assert.Equal(t, "503 Service Unavailable", e.Status)
		assert.Empty(t, e.Message)
	})
}

func TestErrorKeepsHistoricalText(t *testing.T) {
	body := xoNoSuchObjectBody(testVMID, typeVM)
	e := NewFromResponse(methodGET, testVMURL, http.StatusNotFound, notFound, body)

	assert.Equal(t, fmt.Sprintf("API error: %s - %s", notFound, string(body)), e.Error())

	// A typed-nil *Error must not panic when formatted.
	var nilErr *Error
	assert.Equal(t, "API error: <nil>", nilErr.Error())
}

func TestAs(t *testing.T) {
	e := NewFromResponse(methodGET, testVMURL, http.StatusNotFound, notFound, nil)
	wrapped := fmt.Errorf("vm service: %w", fmt.Errorf("get: %w", e))

	got, ok := As(wrapped)
	require.True(t, ok)
	assert.Same(t, e, got)

	_, ok = As(errors.New("plain"))
	assert.False(t, ok)

	_, ok = As(nil)
	assert.False(t, ok)

	// A typed-nil *Error in a non-nil interface must not report ok.
	var nilErr *Error
	_, ok = As(fmt.Errorf("wrap: %w", nilErr))
	assert.False(t, ok)

	// NoSuchObject is nil-safe.
	_, ok = nilErr.NoSuchObject()
	assert.False(t, ok)
}

func TestIsNotFound(t *testing.T) {
	assert.True(t, IsNotFound(NewFromResponse(methodGET, testVMURL, http.StatusNotFound, notFound, []byte(htmlBody))))
	plain404 := NewFromResponse(methodGET, testVMURL, http.StatusNotFound, notFound, nil)
	assert.True(t, IsNotFound(fmt.Errorf("wrapped: %w", plain404)))
	assert.False(t, IsNotFound(NewFromResponse(methodGET, testVMURL, http.StatusInternalServerError, statusText, nil)))
	assert.False(t, IsNotFound(errors.New("API error: 404 Not Found - text only")))
	assert.False(t, IsNotFound(nil))

	// A typed-nil *Error in the chain must not panic and must report false.
	var nilErr *Error
	assert.False(t, IsNotFound(fmt.Errorf("wrap: %w", nilErr)))
}

func TestIsNoSuchObject(t *testing.T) {
	newErr := func(status int, body []byte) error {
		return NewFromResponse(methodGET, testVMURL, status, http.StatusText(status), body)
	}
	xo404 := func(id, typ string) error {
		return newErr(http.StatusNotFound, xoNoSuchObjectBody(id, typ))
	}

	tests := []struct {
		name     string
		err      error
		id       string // defaults to testVMID
		expected bool
	}{
		{name: "XO 404 for the VM", err: xo404(testVMID, typeVM), expected: true},
		{name: "XO 404 without type", err: xo404(testVMID, ""), expected: true},
		{name: "type with another case", err: xo404(testVMID, "vm"), expected: false},
		{name: "upper-case id", err: xo404(testVMID, typeVM), id: strings.ToUpper(testVMID), expected: true},
		{name: "wrapped with %w", err: fmt.Errorf("get vm: %w", xo404(testVMID, typeVM)), expected: true},
		{name: "another id", err: xo404(otherVMID, typeVM), expected: false},
		{name: "another type", err: xo404(testVMID, "pool"), expected: false},
		{
			name:     "XO error without data",
			err:      newErr(http.StatusNotFound, []byte(`{"error":"no such VM `+testVMID+`"}`)),
			expected: false,
		},
		{name: "HTML 404", err: newErr(http.StatusNotFound, []byte(htmlBody)), expected: false},
		{
			name:     "500 with XO body",
			err:      newErr(http.StatusInternalServerError, xoNoSuchObjectBody(testVMID, typeVM)),
			expected: false,
		},
		{
			name:     "text only",
			err:      fmt.Errorf("API error: 404 Not Found - %s", xoNoSuchObjectBody(testVMID, typeVM)),
			expected: false,
		},
		{name: "nil", err: nil, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.id
			if id == "" {
				id = testVMID
			}
			assert.Equal(t, tt.expected, IsNoSuchObject(tt.err, payloads.ResourceTypeVM, id))
		})
	}
}
