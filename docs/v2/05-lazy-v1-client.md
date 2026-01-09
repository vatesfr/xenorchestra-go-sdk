# V1Client() Call Schema - Lazy Initialization

We can provide access to the v1 client directly, allowing users to:
	1. Access v1 functionality without initializing a separate client
	2. Use v2 features while maintaining backward compatibility
	3. Gradually migrate from v1 to v2 without managing multiple clients

## Overview

The v2 SDK client can now be created without requiring an active Xen Orchestra connection. 
The v1 client is initialized lazily on first access, either via:
1. Calling `V1Client()` directly
2. Making a JSON-RPC call through the internal service

This allows v2 client creation without requiring an active XOA connection at initialization time.
This also avoid to open a websocket connection if the user only uses REST API.

## Call Timing

```
Timeline of Execution:

1. v2.New(config) called
   ├─> Creates v2.XOClient{} with REST (v2) services
   ├─> Prepares v1Config for later use
   ├─> Creates jsonrpc.LazyService with factory=initV1Client
   └─> Returns immediately - NO v1 connection attempt yet!

2. User calls: xoClient.V1Client()
   ├─> Calls initV1Client() (first time)
   ├─> sync.Once guard activates
   ├─> v1.NewClient(v1Config) executed
   │   ├─> Tries WebSocket connection to XOA
   │   ├─> Succeeds → v1Client cached, v1InitErr = nil
   │   └─> Fails → v1InitErr cached, v1Client = nil
   └─> Returns (possibly nil if connection failed)

3. User calls: xoClient.V1Client() again
   ├─> Calls initV1Client()
   ├─> sync.Once guard skips (already executed)
   ├─> Returns cached v1Client immediately
   └─> No new connection attempt!

4. User calls: jsonrpcSvc.Call(method, params, &result)
   ├─> LazyService.Call() method invoked
   ├─> factoryOnce.Do() activates (first time)
   ├─> Calls xoClient.initV1Client()
   │   └─> Reuses v1Client if already initialized
   │   └─> Or creates it for the first time
   ├─> If initialization failed: returns error
   └─> If successful: proceeds with JSON-RPC call
```

## Code Locations
- **v2/xo.go**: XOClient with lazy v1 initialization
  - `New()` - creates XOClient without initializing v1
  - `initV1Client()` - lazy factory for v1 client (sync.Once guarded)
  - `V1Client()` - getter that triggers lazy init

- **pkg/services/jsonrpc/service.go**: JSONRPC service implementations
  - `Service` - standard service with eager client
  - `LazyService` - embeds Service, defers initialization
  - `NewLazy()` - creates LazyService with factory function

## Implementation Benefits

✅ **Non-blocking client creation**: v2 client ready immediately, no network calls at init time  
✅ **No more unused websocket**: This avoid to open a websocket connection if the user only uses REST API.  
✅ **Backward compatible**: Existing v1 client functionality unchanged, still accessible  
✅ **Thread-safe**: sync.Once guarantees single execution, all goroutines share same instance  

## Usage Patterns

```go
// Pattern 1: Create v2 client (v1 NOT initialized)
cfg := &config.Config{
    Url:   "http://xoa:80",
    Token: "token123",
}
xoClient, err := v2.New(cfg)
if err != nil {
    panic(err)
}
// At this point:
// - REST v2 services ready immediately
// - v1 client NOT created yet
// - No network connection to XOA required


// Pattern 2: Explicit v1 client access (triggers lazy init)
v1 := xoClient.V1Client()
if v1 != nil {
    // Use v1 client for features not yet in v2
    result := v1.GetUser(client.User{Email: "golang-client-test"})
}
// If initialization fails, v1 will be nil
// Error details accessible via xoClient.v1InitErr

// Pattern 3: Detect initialization failures
v1 := xoClient.V1Client()
if xoClient.v1InitErr != nil {
    log.Printf("v1 client failed to initialize: %v", xoClient.v1InitErr)
    // only REST v2 services are available
}

```
