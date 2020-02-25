package jsonrpc2

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github/brianso/go-jsonrpc2"
	"os"
)

func main() {
	server := jsonrpc2.NewServer()
	server.DefineMethod("echo", func(ctx context.Context, params json.RawMessage) (result interface{}, error error) {
		return params, nil
	})
	server.DefineMethod("add", func(ctx context.Context, params json.RawMessage) (result interface{}, error error) {
		var p [2]float64
		json.Unmarshal(params, &p)
		return p[0] + p[1], nil
	})
	rsp := server.ServeRequest(json.RawMessage(`{ "jsonrpc": "2.0", "method": "echo", "params": "hi", "id": 1 }`))
	fmt.Printf("response = %s\n", rsp)
	rsp = server.ServeRequest(json.RawMessage(`[{ "jsonrpc": "2.0", "method": "add", "params": [1,2], "id": 1 }]`))
	fmt.Printf("response = %s\n", rsp)
	rsp = server.ServeRequest(json.RawMessage(`[{ "jsonrpc": "2.0", "method": "add", "params": [1,2] }]`))
	fmt.Printf("response = %s\n", rsp)

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter json: ")
		jsonStr, _ := reader.ReadString('\n')
		resp := server.ServeRequest(json.RawMessage(jsonStr))
		os.Stdout.Write(resp)
		os.Stdout.Write([]byte{'\n'})
	}
}