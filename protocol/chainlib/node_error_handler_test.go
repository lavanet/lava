package chainlib

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/utils"
	"github.com/stretchr/testify/require"
)

func TestNodeErrorHandlerGenericErrors(t *testing.T) {
	ctx := context.Background()
	neh := &genericErrorHandler{}

	// Test context.DeadlineExceeded error
	withTimeout, cancel := context.WithTimeout(ctx, 0)
	defer cancel()
	err := neh.handleGenericErrors(withTimeout, nil)
	expectedError := utils.LavaFormatError("Provider Failed Sending Message", context.DeadlineExceeded)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test net.ErrWriteToConnected error
	err = neh.handleGenericErrors(ctx, net.ErrWriteToConnected)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Write to connected connection", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test net.ErrClosed error
	err = neh.handleGenericErrors(ctx, net.ErrClosed)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Operation on closed connection", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test io.EOF error
	err = neh.handleGenericErrors(ctx, io.EOF)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: End of input stream reached", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test net.OpError with timeout error
	opErr := &net.OpError{
		Op:     "dummy",
		Net:    "dummy",
		Source: nil,
		Addr:   nil,
		Err:    os.ErrDeadlineExceeded,
	}
	err = neh.handleGenericErrors(ctx, opErr)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Network operation timed out", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test net.DNSError error
	dnsErr := &net.DNSError{
		Err: "dummy",
	}
	err = neh.handleGenericErrors(ctx, dnsErr)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: DNS resolution failed", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test net.OpError with connection refused error
	opErr = &net.OpError{
		Op:     "dummy",
		Net:    "dummy",
		Source: nil,
		Addr:   nil,
		Err: &os.SyscallError{
			Syscall: "dummy",
			Err:     syscall.ECONNREFUSED,
		},
	}
	err = neh.handleGenericErrors(ctx, opErr)
	expectedError = utils.LavaFormatError("Provider Side Failed Sending Message, Reason: Connection refused", nil)
	require.Equal(t, err.Error(), expectedError.Error())

	// Test non-matching error
	err = neh.handleGenericErrors(ctx, errors.New("dummy error"))
	require.Equal(t, err, nil)
}

func TestNodeErrorHandlerTimeout(t *testing.T) {
	httpClient := &http.Client{
		Timeout: 5 * time.Minute, // we are doing a timeout by request
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	msgBuffer := bytes.NewBuffer([]byte{1, 2, 3})
	req, err := http.NewRequestWithContext(ctx, "test", "http://0.0.0.0:6789", msgBuffer)
	require.NoError(t, err)
	_, err = httpClient.Do(req)
	require.Error(t, err)
	utils.LavaFormatDebug(err.Error())
	genericHandler := genericErrorHandler{}
	bctx := context.Background()
	ret := genericHandler.handleGenericErrors(bctx, err)
	utils.LavaFormatDebug(ret.Error())
	require.NotContains(t, ret.Error(), "http://0.0.0.0:6789")
}

func TestNodeErrorJQ(t *testing.T) {
	handler := &genericErrorHandler{}

	malformedJSON := `[
  [
    {
      "inner": "0xabed5d742e4de942f5b1b4e419ff8b51a5361c5ab0cbbe59824a5c4249e426f"
    },
    {
      "inner": "0x78f3de4897720dbf03ada5c018c35a5ea41f9351bb5a80405091d18bb4fe7d1a"
    },
    {
      "inner": "0xf9b3b3e0f56612d0066a64f512f6c109df471806ab199c60dfaf258450ca85cd"
    },
    {
      "inner": "0x2cd81d6cab1f3bc4750389aa62107f7507beeadb9090b902e6c9f212eba2f26a"
    },
    {
      "inner": "0x90a5a36890cb9fd19ebe19b5c966bef698f51f09e05936b84e9cf11105f127da"
    },
    {
      "inner": "0x5bcf3785fcecfcbdbbededfe4b8c816f42ba0cffc545b8e44155bb62a74c87ea"
    },
    {
      "inner": "0xf12578c6f7401062e7e87472bd56b8d09c635eba29ea074b0b9462382d02bdf0"
    },
    {
      "inner": "0xec6e9bfd7cc4b4634ba6a127cc122587ae294385e51e074a43fd3d8f072e412b"
    },
    {
      "inner": "0xa14e466ac871f3cda148fd7621d1b312ffcee218173b0bf68175160af223f721"
    },
    {
      "inner": "0x26ec0ab0d408ac512cf300746ea3563d9b103feda231a067db2baf1259b35883"
    },
    {
      "inner": "0x7bdfe2dbfd46141497229b6d81279e130ba3f8252563d59ea9c6cf26cb685405"
    },
    {
      "inner": "0x11139df1af7051a26114d7aeee6dd4ac9eb31ff7785c297fcfc8af218bb48dd6"
    },
    {
      "inner": "0xb89df2c717c8aec4658a38427a4adf9c30ad3e899f0f4259804bad3e59c20051"
    },
    {
      "inner": "0xb389f719b6cd4e2835b5115127209f8dbf80dd8534a94399e81c9cf6f48c6245"
    },
    {
      "inner": "0xa0454b18a94f9d0276176ba3ceba900fb28d179aa9bc276c07b2116b5e6d962a"
    },
    {
      "inner": "0xc24fd6702b84b2524b3c708a48524297294ed022d9c2f30222a743ef3773b2b9"
    },
    {
      "inner": "0xea565dcc843495a292dad0249d4871ce0d39651c65b3442101903fc6dbf7def8"
    },
    {
      "inner": "0x4525e62437f8928b5d68e2a239861e609b229f887c21ad3e4617e13eaa6c67ae"
    },
    {
      "inner": "0x2aa729ca8fdc4ff4150d6dc20061c219506b3c2cefeb96b9516790f09da89a82"
    },
    {
      "inner": "0xec9b7452456d291644384643307c8b883b4398acb448746d006a3bfc41e1b2b4"
    },
    {
      "inner": "0x7c6a5f19c2404d8370535ad9f4d06fe87adf9dfde116355dfcc59d5bf0d81587"
    },
    {
      "inner": "0xd8db10dd9de4a5a8c62c4e690428e5acf0c0c31a4bc507f74a26b22529a5739c"
    }
  ]
]`

	err := handler.HandleJSONFormatError([]byte(malformedJSON))
	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	}
}
