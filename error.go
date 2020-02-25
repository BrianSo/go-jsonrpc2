package jsonrpc2

// Rpc Error
// You may return by `jsonrpc2.NewError(code, msg)`. This will be used in the error response.
type Error interface {
	Code() int
	Error() string
}

var (
	ErrParseError = rpcError{
		ErrorCode:    -32700,
		Message: "Parse error",
	}
	ErrInvalidRequest = rpcError{
		ErrorCode:    -32600,
		Message: "Invalid request",
	}
	ErrMethodNotFound = rpcError{
		ErrorCode:    -32601,
		Message: "Method not found",
	}
	ErrInvalidParams = rpcError{
		ErrorCode:    -32602,
		Message: "Invalid Params",
	}
)

func NewError(code int, msg string) Error {
	return &rpcError{
		ErrorCode: code,
		Message: msg,
	}
}

func NewInternalError(msg string) Error {
	return NewError(-32000, msg)
}

// ============ Private members below =================

type rpcError struct {
	ErrorCode   int    `json:"code"`
	Message 	string `json:"message"`
}

func (e rpcError) Error() string {
	return e.Message
}

func (e rpcError) Code() int {
	return e.ErrorCode
}