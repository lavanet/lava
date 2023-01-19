package utils_test

import (
	"errors"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	"github.com/stretchr/testify/require"
)

var (
	TestError = sdkerrors.New("test Error", 123, "error for tests")
)

// TestErrorTypeChecks tests if type does not change after LavaFormatError
func TestErrorTypeChecks(t *testing.T) {
	var err error = TestError
	newErr := utils.LavaFormatError("testing 123", err, &map[string]string{"attribute": "test"})
	require.True(t, TestError.Is(newErr))
}

// TestErrorCodeChecks tests if error code does not change after LavaFormatError
func TestErrorCodeChecks(t *testing.T) {
	var err error = TestError
	newErr := utils.LavaFormatError("testing 123", err, &map[string]string{"attribute": "test"})

	// Extract the code from wrapped error
	_, code, _ := sdkerrors.ABCIInfo(newErr, false)

	// Make sure the code is same
	require.Equal(t, TestError.ABCICode(), code)

	// Create new error with standard package
	err = errors.New("test err")

	newErr = utils.LavaFormatError("testing 123", err, &map[string]string{"attribute": "test"})

	// Extract the code from wrapped error
	_, code, _ = sdkerrors.ABCIInfo(newErr, false)

	// For all errors created with standard go package
	// error code will be 1
	require.Equal(t, uint32(1), code)
}
