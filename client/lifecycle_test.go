package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/sourcegraph/jsonrpc2/websocket"
)

// fakeXO is a minimal JSON-RPC over websocket server mimicking the XO
// endpoints used by the client lifecycle (sign in, token create/delete).
type fakeXO struct {
	server *httptest.Server

	signInErr      *jsonrpc2.Error
	tokenCreateErr *jsonrpc2.Error

	mu            sync.Mutex
	calls         []string
	deletedTokens []string
	disconnected  chan struct{}
}

const fakeCreatedToken = "sdk-created-token"

func newFakeXO(t *testing.T) *fakeXO {
	t.Helper()
	f := &fakeXO{disconnected: make(chan struct{}, 1)}
	upgrader := gorillawebsocket.Upgrader{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		conn := jsonrpc2.NewConn(context.Background(), websocket.NewObjectStream(ws),
			jsonrpc2.HandlerWithError(f.handle))
		<-conn.DisconnectNotify()
		f.disconnected <- struct{}{}
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeXO) handle(_ context.Context, _ *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req.Method)

	switch req.Method {
	case "session.signIn":
		if f.signInErr != nil {
			return nil, f.signInErr
		}
		return signInResponse{Id: "user-id"}, nil
	case "token.create":
		if f.tokenCreateErr != nil {
			return nil, f.tokenCreateErr
		}
		return fakeCreatedToken, nil
	case "token.delete":
		var params struct {
			Token string `json:"token"`
		}
		if req.Params != nil {
			_ = json.Unmarshal(*req.Params, &params)
		}
		f.deletedTokens = append(f.deletedTokens, params.Token)
		return true, nil
	}
	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: req.Method}
}

func (f *fakeXO) config() Config {
	return Config{Url: "ws" + strings.TrimPrefix(f.server.URL, "http")}
}

func (f *fakeXO) deleted() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.deletedTokens...)
}

func (f *fakeXO) waitDisconnect(t *testing.T) {
	t.Helper()
	select {
	case <-f.disconnected:
	case <-time.After(5 * time.Second):
		t.Fatal("websocket connection was not closed")
	}
}

func TestNewClient_closesConnectionWhenSignInFails(t *testing.T) {
	f := newFakeXO(t)
	f.signInErr = &jsonrpc2.Error{Code: 3, Message: "invalid credentials"}

	cfg := f.config()
	cfg.Token = "bad-token"
	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected sign in error")
	}

	f.waitDisconnect(t)
}

func TestNewClient_closesConnectionWhenTokenCreateFails(t *testing.T) {
	f := newFakeXO(t)
	f.tokenCreateErr = &jsonrpc2.Error{Code: 2, Message: "not enough permissions"}

	cfg := f.config()
	cfg.Username = "user"
	cfg.Password = "pass"
	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected token.create error")
	}

	f.waitDisconnect(t)
	if got := f.deleted(); len(got) != 0 {
		t.Errorf("no token should be revoked when none was created, got %v", got)
	}
}
