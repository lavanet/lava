package utils_test

import (
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	"github.com/stretchr/testify/require"
)

var TestError = sdkerrors.New("test Error", 123, "error for tests")

func TestErrorTypeChecks(t *testing.T) {
	var err error = TestError
	newErr := utils.LavaFormatError("testing 123", err, utils.Attribute{"attribute", "test"})
	require.True(t, TestError.Is(newErr))
}
