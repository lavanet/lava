package performance

import "errors"

var (
	NotConnectedError   = errors.New("No Connection To grpc server")
	NotInitializedError = errors.New("to use cache run initCache")
)
