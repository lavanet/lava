package common

import "github.com/gofiber/fiber/v2"

// #######
// JsonRPC
// #######

type JsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
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

var JsonRpcRateLimitError = JsonRPCErrorMessage{
	JsonRPC: "2.0",
	Id:      1,
	Error: JsonRPCError{
		Code:    429,
		Message: "Too Many Requests",
	},
}

var JsonRpcParseError = JsonRPCErrorMessage{
	JsonRPC: "2.0",
	Id:      -1,
	Error: JsonRPCError{
		Code:    -32700,
		Message: "Parse error",
		Data:    "Failed to parse the request body as JSON",
	},
}

var JsonRpcSubscriptionNotFoundError = JsonRPCErrorMessage{
	JsonRPC: "2.0",
	Id:      1,
	Error: JsonRPCError{
		Code:    -32603,
		Message: "Internal error",
		Data:    "subscription not found",
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

func CreateRestMethodNotFoundError(fiberCtx *fiber.Ctx, chainId string) error {
	switch chainId {
	case "APT1":
		// Aptos node returns a different error body than the rest of the chains
		// This solution is temporary until we change the spec to state how the error looks like
		return fiberCtx.Status(fiber.StatusNotImplemented).JSON(RestAptosMethodNotFoundError)
	default:
		return fiberCtx.Status(fiber.StatusNotImplemented).JSON(RestMethodNotFoundError)
	}
}
