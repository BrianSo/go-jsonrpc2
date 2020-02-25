package jsonrpc2

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestServer_ServeRequest(t *testing.T) {
	server := NewServer()
	server.DefineMethod("echo", func(ctx context.Context, params json.RawMessage) (result interface{}, error error) {
		return struct {
			EchoResult interface{}
		}{EchoResult: params}, nil
	})
	t.Run("id is integer", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`{ "jsonrpc": "2.0", "method": "echo", "params": "hi", "id": 1 }`))
		require.JSONEq(t, `{
			"id": 1,
			"jsonrpc": "2.0",
			"result": { "EchoResult": "hi" }
		}`, string(rsp))
	})
	t.Run("id is string", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`{ "jsonrpc": "2.0", "method": "echo", "params": "hi", "id": "1" }`))
		require.JSONEq(t, `{
			"id": "1",
			"jsonrpc": "2.0",
			"result": { "EchoResult": "hi" }
		}`, string(rsp))
	})
	t.Run("is notification", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`{ "jsonrpc": "2.0", "method": "echo", "params": "hi" }`))
		require.Equal(t, "", string(rsp))
	})
	t.Run("method is not defined", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`{ "jsonrpc": "2.0", "method": "no way", "params": "hi", "id": 1 }`))
		require.JSONEq(t, `{
			"id": 1,
			"jsonrpc": "2.0",
			"error": {"code": -32601, "message": "Method not found"}
		}`, string(rsp))
	})
	t.Run("invalid json", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`{qqqq}`))
		require.JSONEq(t, `{
			"id": null,
			"jsonrpc": "2.0",
			"error": {"code": -32700, "message": "Parse error"}
		}`, string(rsp))
	})
	t.Run("invalid request", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`{"a": 1}`))
		require.JSONEq(t, `{
			"id": null,
			"jsonrpc": "2.0",
			"error": {"code": -32600, "message": "Invalid request"}
		}`, string(rsp))
	})
}

func TestServer_ServeRequestWithTimeout(t *testing.T) {
	server := NewServer()
	server.SetDefaultTimeout(5 * time.Millisecond)
	server.DefineMethod("wait", func(ctx context.Context, params json.RawMessage) (result interface{}, error error) {
		var n float64
		json.Unmarshal(params, &n)
		time.Sleep(time.Duration(int(n)) * time.Millisecond)
		return "ok", nil
	})
	t.Run("should not timeout", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`{ "jsonrpc": "2.0", "method": "wait", "params": 1, "id": "1" }`))
		require.JSONEq(t, `{
			"id": "1",
			"jsonrpc": "2.0",
			"result": "ok"
		}`, string(rsp))
	})
	t.Run("should timeout", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`{ "jsonrpc": "2.0", "method": "wait", "params": 100, "id": "1" }`))
		require.JSONEq(t, `{
			"id": "1",
			"jsonrpc": "2.0",
			"error": { "code": -32000, "message":"context deadline exceeded" }
		}`, string(rsp))
	})
}

func TestServer_ServeBatchRequest(t *testing.T) {
	server := NewServer()
	server.SetDefaultTimeout(5 * time.Millisecond)
	server.DefineMethod("echo", func(ctx context.Context, params json.RawMessage) (result interface{}, error error) {
		return struct {
			EchoResult interface{}
		}{EchoResult: params}, nil
	})
	server.DefineMethod("wait", func(ctx context.Context, params json.RawMessage) (result interface{}, error error) {
		var n float64
		json.Unmarshal(params, &n)
		time.Sleep(time.Duration(int(n)) * time.Millisecond)
		return "ok", nil
	})
	t.Run("success", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`[{ "jsonrpc": "2.0", "method": "echo", "params": "hi", "id": 1 }]`))
		require.JSONEq(t, `[{
			"id": 1,
			"jsonrpc": "2.0",
			"result": { "EchoResult": "hi" }
		}]`, string(rsp))
	})
	t.Run("should invalid request given empty error", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`[]`))
		require.JSONEq(t, `{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid request"}, "id": null}`, string(rsp))
	})
	t.Run("should invalid request individually", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`[
			1, 
			{ "method": "echo" }, 
			{ "jsonrpc": "2.0", "method": "echo", "params": "hi", "id": "1" }
		]`))
		require.JSONEq(t, `[
			{"jsonrpc": "2.0", "error": {"code": -32700, "message": "Parse error"}, "id": null},
			{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid request"}, "id": null},
			{"jsonrpc": "2.0", "result": { "EchoResult": "hi" }, "id": "1"}
		]`, string(rsp))
	})
	t.Run("should 1 timeout and 1 success", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`[
			{ "jsonrpc": "2.0", "method": "wait", "params": 100, "id": "1" },
			{ "jsonrpc": "2.0", "method": "wait", "params": 1, "id": "2" }
		]`))
		require.JSONEq(t, `[{
			"id": "1",
			"jsonrpc": "2.0",
			"error": { "code": -32000, "message":"context deadline exceeded" }
		}, {
			"id": "2",
			"jsonrpc": "2.0",
			"result": "ok"
		}]`, string(rsp))
	})
	t.Run("1 notification and 1 success", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`[
			{ "jsonrpc": "2.0", "method": "echo", "params": 100 },
			{ "jsonrpc": "2.0", "method": "echo", "params": 1, "id": "2" }
		]`))
		require.JSONEq(t, `[{
			"id": "2",
			"jsonrpc": "2.0",
			"result": { "EchoResult": 1 }
		}]`, string(rsp))
	})
	t.Run("all notifications ", func(t *testing.T) {
		rsp := server.ServeRequest(json.RawMessage(`[
			{ "jsonrpc": "2.0", "method": "echo", "params": 100 },
			{ "jsonrpc": "2.0", "method": "echo", "params": 1 }
		]`))
		require.Equal(t, "", string(rsp))
	})
}
