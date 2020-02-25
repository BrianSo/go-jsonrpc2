# go-jsonrpc2
This is a transport independent jsonrpc 2.0 implementation.

## Usage
```go
server := jsonrpc2.NewServer()
server.DefineMethod("echo", func(params interface{}) (result interface{}, error error) {
    return params, nil
})
server.DefineMethod("add", func(params interface{}) (result interface{}, error error) {
    p := params.([]interface{})
    return p[0].(float64) + p[1].(float64), nil
})
rsp := server.ServeRequest(json.RawMessage(`{ "jsonrpc": "2.0", "method": "echo", "params": "hi", "id": 1 }`))
fmt.Printf("response = %s\n", rsp)
// output: response = {"id":1,"jsonrpc":"2.0","result":"hi"}

rsp = server.ServeRequest(json.RawMessage(`[{ "jsonrpc": "2.0", "method": "add", "params": [1,2], "id": 1 }]`))
fmt.Printf("response = %s\n", rsp)
// output: response = [{"id":1,"jsonrpc":"2.0","result":3}]

// notification
rsp = server.ServeRequest(json.RawMessage(`[{ "jsonrpc": "2.0", "method": "add", "params": [1,2] }]`))
fmt.Printf("response = %s\n", rsp)
// output: response = 
// when rsp is nil, it is an notification request, no need to send response.
```

### Error handling
You may return `jsonrpc2.Error` in Handler.
```go
server.DefineMethod("echo", func(params interface{}) (result interface{}, error error) {
    return nil, jsonrpc2.NewError(-32001, "My Custom Error")
})
```
--> `{"jsonrpc":"2.0","error":{"code":-32001,message:"My Custom Error"},id:<RREQUEST_ID>}`

if a normal error is returned, `code: -32000` is used