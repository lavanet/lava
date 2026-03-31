package utils_test

import (
	"errors"
	"testing"

	"github.com/lavanet/lava/v5/utils"
	"github.com/stretchr/testify/require"
)

var TestError = errors.New("error for tests")

func TestErrorTypeChecks(t *testing.T) {
	var err error = TestError
	newErr := utils.LavaFormatError("testing 123", err, utils.Attribute{"attribute", "test"})
	require.True(t, errors.Is(newErr, TestError))
}
