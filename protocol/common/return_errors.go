package common

// #######
// JsonRPC
// #######

type JsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type JsonRPCErrorMessage struct {
	JsonRPC string       `json:"jsonrpc"`
	Id      int          `json:"id"`
	Error   JsonRPCError `json:"error"`
}

var JsonRpcMethodNotFoundError = JsonRPCErrorMessage{
	JsonRPC: "2.0",
	Id:      1,
	Error: JsonRPCError{
		Code:    -32601,
		Message: "Method not found",
	},
}

// #######
// Rest
// #######

type RestError struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Details []interface{} `json:"details"`
}

var RestMethodNotFoundError = RestError{
	Code:    12,
	Message: "Not Implemented",
	Details: []interface{}{},
}

// #######
// Rest - Aptos
// #######

type RestAptosError struct {
	Message     string      `json:"message"`
	ErrorCode   string      `json:"error_code"`
	VmErrorCode interface{} `json:"vm_error_code"`
}

var RestAptosMethodNotFoundError = RestAptosError{
	Message:     "not found",
	ErrorCode:   "web_framework_error",
	VmErrorCode: nil,
}
