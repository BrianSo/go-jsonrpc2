// jsonrpc2 provides a transport independent server implementation
package jsonrpc2

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type (

	// Usage:
	//	server := jsonrpc2.NewServer()
	//	server.DefineMethod("echo", func(params interface{}) (result interface{}, error error) {
	//		return params, nil
	//	})
	//
	// rsp, err := server.ServeRequest(`{ "jsonrpc": "2.0", "method": "echo", "params": "hi", "id": 1 }`)
	// // rsp is a json string in jsonrpc2.0 format
	// // send your rsp through your transport (e.g. http)
	Server interface{
		SetDefaultTimeout(timeout time.Duration)
		DefineMethod(method string, h Handler)
		ServeRequest(jsonString json.RawMessage) json.RawMessage
	}


	// The handler of your server methods. If error returned is jsonrpc2.Error, the code will be used.
	Handler func(ctx context.Context, params json.RawMessage) (result interface{}, error error)
)

func NewServer() Server {
	return &server{
		handlers: map[string]Handler{},
		timeout:  0,
	}
}

// ============ Private members below =================

type (
	server struct {
		handlers map[string]Handler
		timeout  time.Duration
	}

	// A request represents a JSON-RPC request received by the server.
	request struct {
		ID      json.RawMessage `json:"id"`
		Version string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}

	// A response represents a JSON-RPC Resp returned by the server.
	response struct {
		ID      json.RawMessage `json:"id"`
		Version string          `json:"jsonrpc"`
		Result  interface{}     `json:"result,omitempty"`
		Error   Error          `json:"error,omitempty"`
	}
)

func (s *server) SetDefaultTimeout(timeout time.Duration) {
	s.timeout = timeout
}

func (s *server) DefineMethod(method string, h Handler) {
	s.handlers[method] = h
}

// Receive a jsonrpc 2.0 json string request and return a jsonrpc 2.0 json string response
func (s *server) ServeRequest(jsonString json.RawMessage) json.RawMessage {
	var arr []json.RawMessage
	if err := json.Unmarshal(jsonString, &arr); err == nil {
		if len(arr) == 0 {
			return makeResponseJson(request{}, nil, ErrInvalidRequest)
		}
		return s.serveBatchRequest(arr)
	}
	return s.serveSingleRequest(jsonString)
}

func (s *server) serveSingleRequest(jsonString json.RawMessage) json.RawMessage {
	r := &request{}
	if err := json.Unmarshal(jsonString, r); err != nil {
		return makeResponseJson(request{}, nil, ErrParseError)
	}
	if err := validateRequest(*r); err != nil {
		return makeResponseJson(*r, nil, err)
	}
	h, ok := s.handlers[r.Method]
	if !ok {
		return makeResponseJson(*r, nil, ErrMethodNotFound)
	}
	ctx := context.Background()
	if s.timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}
	result, err := handleAsync(ctx, h, r.Params)
	return makeResponseJson(*r, result, err)
}

func (s *server) serveBatchRequest(rs []json.RawMessage) json.RawMessage {
	rsps := make([]json.RawMessage, len(rs))
	var wg sync.WaitGroup
	for i := range rs {
		wg.Add(1)
		go func(i int) {
			rsps[i] = s.serveSingleRequest(rs[i])
			wg.Done()
		}(i)
	}
	wg.Wait()

	// construct response
	result := make([]json.RawMessage, 0)
	for i := range rsps {
		if rsps[i] != nil {
			result = append(result, rsps[i])
		}
	}
	if len(result) == 0 {
		return nil
	}
	rsp, _ := json.Marshal(result)
	return rsp
}

// Rpc Handler is called with a timeout timer. If timed out, throw context deadline exceed error
func handleAsync(ctx context.Context, h Handler, params json.RawMessage) (resp interface{}, err error) {
	// no timeout
	deadline, ok := ctx.Deadline()
	if !ok {
		return h(ctx, params)
	}

	// with timeout
	done := make(chan int)

	// timeout timer
	go func() {
		timeout := deadline.Sub(time.Now())
		if timeout > 0 {
			time.Sleep(timeout)
		}
		err = context.DeadlineExceeded
		done <- 1
	}()

	// main handler
	go func() {
		resp, err = h(ctx, params)
		done <- 1
	}()

	// wait for 1 of the goroutine finish
	<-done
	return resp, err
}

func validateRequest(req request) error {
	if req.Version != "2.0" {
		return ErrInvalidRequest
	}
	if req.Method == "" {
		return ErrInvalidRequest
	}
	return nil
}

func makeResponseJson(request request, result interface{}, error error) json.RawMessage {
	// if notification request
	if validateRequest(request) == nil && request.ID == nil {
		return nil
	}
	r := response{
		ID:      request.ID,
		Version: "2.0",
	}
	if error != nil {
		if e, ok := error.(Error); ok {
			// reconstruct to use private rpcError for json.Marshall
			r.Error = NewError(e.Code(), e.Error())
		} else {
			r.Error = NewInternalError(error.Error())
		}
	}
	r.Result = result
	respStr, _ := json.Marshal(r)
	return respStr
}
