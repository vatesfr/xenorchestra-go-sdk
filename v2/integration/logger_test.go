package integration

import (
	"fmt"
	"net/url"
	"sync"
	"testing"

	"go.uber.org/zap"
)

// testLogWriter is a custom io.Writer that writes logs to testing.T.Logf
type testLogWriter struct {
	t *testing.T
}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	w.t.Logf("%s", p)
	return len(p), nil
}

// testSink is a custom zap sink that writes logs to testing.T.Logf
type testSink struct {
	testName string
}

func (s *testSink) Write(p []byte) (n int, err error) {
	testSinksMu.RLock()
	t, ok := testSinks[s.testName]
	testSinksMu.RUnlock()

	if ok {
		t.Logf("%s", p)
	}
	return len(p), nil
}

func (s *testSink) Sync() error {
	return nil
}

func (s *testSink) Close() error {
	return nil
}

var (
	registerOnce sync.Once
	testSinksMu  sync.RWMutex
	testSinks    = make(map[string]testing.TB)
)

// RegisterTestingSink registers a custom zap sink that outputs to testing.T.
// It returns the connection string that should be used in OutputPaths.
// It is safe for parallel tests.
func RegisterTestingSink(t testing.TB) string {
	name := t.Name()

	testSinksMu.Lock()
	testSinks[name] = t
	testSinksMu.Unlock()

	t.Cleanup(func() {
		testSinksMu.Lock()
		delete(testSinks, name)
		testSinksMu.Unlock()
	})

	registerOnce.Do(func() {
		_ = zap.RegisterSink("test", func(u *url.URL) (zap.Sink, error) {
			return &testSink{
				testName: u.Query().Get("name"),
			}, nil
		})
	})

	return fmt.Sprintf("test:?name=%s", url.QueryEscape(name))
}
