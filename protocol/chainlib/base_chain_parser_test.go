package chainlib

import (
	reflect "reflect"
	"strconv"
	"testing"

	"github.com/lavanet/lava/v4/protocol/chainlib/extensionslib"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestGetVerifications(t *testing.T) {
	verifications := map[VerificationKey]map[string][]VerificationContainer{
		{
			Extension: "",
			Addon:     "",
		}: {
			"/x": {
				{InternalPath: "/x"},
			},
			"": {
				{InternalPath: ""},
			},
		},
		{
			Extension: "",
			Addon:     "addon1",
		}: {
			"/x": {
				{InternalPath: "/x"},
			},
			"": {
				{InternalPath: ""},
			},
		},
		{
			Extension: "ext1",
			Addon:     "addon1",
		}: {
			"/x": {
				{InternalPath: "/x"},
			},
			"": {
				{InternalPath: ""},
			},
		},
		{
			Extension: "ext1",
			Addon:     "",
		}: {
			"/x": {
				{InternalPath: "/x"},
			},
			"": {
				{InternalPath: ""},
			},
		},
	}

	playBook := []struct {
		Extension    string
		Addon        string
		InternalPath string
	}{
		{
			Extension:    "",
			Addon:        "",
			InternalPath: "",
		},
		{
			Extension:    "",
			Addon:        "",
			InternalPath: "/x",
		},
		{
			Extension:    "ext1",
			Addon:        "addon1",
			InternalPath: "",
		},
		{
			Extension:    "ext1",
			Addon:        "addon1",
			InternalPath: "/x",
		},
		{
			Extension:    "",
			Addon:        "addon1",
			InternalPath: "",
		},
		{
			Extension:    "",
			Addon:        "addon1",
			InternalPath: "/x",
		},
		{
			Extension:    "ext1",
			Addon:        "",
			InternalPath: "",
		},
		{
			Extension:    "ext1",
			Addon:        "",
			InternalPath: "/x",
		},
	}

	baseChainParser := BaseChainParser{
		verifications: verifications,
		allowedAddons: map[string]bool{"addon1": true},
	}
	baseChainParser.extensionParser = extensionslib.NewExtensionParser(map[string]struct{}{"ext1": {}}, nil)

	for idx, play := range playBook {
		for _, apiInterface := range []string{spectypes.APIInterfaceJsonRPC, spectypes.APIInterfaceTendermintRPC, spectypes.APIInterfaceRest, spectypes.APIInterfaceGrpc} {
			t.Run("GetVerifications "+strconv.Itoa(idx), func(t *testing.T) {
				var supported []string
				if play.Extension == "" && play.Addon == "" {
					supported = []string{""}
				} else if play.Extension == "" {
					supported = []string{play.Addon}
				} else if play.Addon == "" {
					supported = []string{play.Extension}
				} else {
					supported = []string{play.Extension, play.Addon}
				}

				actualVerifications, err := baseChainParser.GetVerifications(supported, play.InternalPath, apiInterface)
				require.NoError(t, err)

				expectedVerificationKey := VerificationKey{Extension: play.Extension, Addon: play.Addon}
				expectedVerifications := verifications[expectedVerificationKey][play.InternalPath]
				// add the empty addon to the expected verifications
				if play.Addon != "" {
					expectedVerificationKey.Addon = ""
					expectedVerifications = append(expectedVerifications, verifications[expectedVerificationKey][play.InternalPath]...)
				}
				require.True(t, reflect.DeepEqual(expectedVerifications, actualVerifications), "expected: %v, actual: %v", expectedVerifications, actualVerifications)
			})
		}
	}
}
