package chainlib

import (
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDefaultApi(t *testing.T) {
	t.Run("default api", func(t *testing.T) {
		baseProtocolMessage := &BaseProtocolMessage{
			ChainMessage: &baseChainMessageContainer{
				api: &types.Api{
					Name: "Default-API",
					BlockParsing: types.BlockParser{
						ParserFunc: types.PARSER_FUNC_DEFAULT,
					},
				},
			},
		}

		assert.True(t, baseProtocolMessage.IsDefaultApi())
	})

	t.Run("non-default api - Name is not Default-*", func(t *testing.T) {
		baseProtocolMessage := &BaseProtocolMessage{
			ChainMessage: &baseChainMessageContainer{
				api: &types.Api{
					Name: "API",
					BlockParsing: types.BlockParser{
						ParserFunc: types.PARSER_FUNC_DEFAULT,
					},
				},
			},
		}

		assert.False(t, baseProtocolMessage.IsDefaultApi())
	})

	t.Run("non-default api - BlockParsing is not default", func(t *testing.T) {
		baseProtocolMessage := &BaseProtocolMessage{
			ChainMessage: &baseChainMessageContainer{
				api: &types.Api{
					Name: "Default-API",
					BlockParsing: types.BlockParser{
						ParserFunc: types.PARSER_FUNC_EMPTY,
					},
				},
			},
		}

		assert.False(t, baseProtocolMessage.IsDefaultApi())
	})
}

func TestGetCrossValidationParameters(t *testing.T) {
	t.Run("no headers present - returns defaults with headersPresent=false", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{},
		}

		params, headersPresent, err := bpm.GetCrossValidationParameters()
		require.NoError(t, err)
		assert.False(t, headersPresent)
		assert.Equal(t, common.DefaultCrossValidationParams, params)
	})

	t.Run("nil headers - returns defaults with headersPresent=false", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: nil,
		}

		params, headersPresent, err := bpm.GetCrossValidationParameters()
		require.NoError(t, err)
		assert.False(t, headersPresent)
		assert.Equal(t, common.DefaultCrossValidationParams, params)
	})

	t.Run("valid headers - parses correctly", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS:    "5",
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "3",
			},
		}

		params, headersPresent, err := bpm.GetCrossValidationParameters()
		require.NoError(t, err)
		assert.True(t, headersPresent)
		assert.Equal(t, 5, params.MaxParticipants)
		assert.Equal(t, 3, params.AgreementThreshold)
	})

	t.Run("threshold equals max - valid", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS:    "3",
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "3",
			},
		}

		params, headersPresent, err := bpm.GetCrossValidationParameters()
		require.NoError(t, err)
		assert.True(t, headersPresent)
		assert.Equal(t, 3, params.MaxParticipants)
		assert.Equal(t, 3, params.AgreementThreshold)
	})

	t.Run("only max-participants header - error", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS: "5",
			},
		}

		_, headersPresent, err := bpm.GetCrossValidationParameters()
		require.Error(t, err)
		assert.True(t, headersPresent)
		assert.Contains(t, err.Error(), "agreement-threshold header is required")
	})

	t.Run("only agreement-threshold header - error", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "3",
			},
		}

		_, headersPresent, err := bpm.GetCrossValidationParameters()
		require.Error(t, err)
		assert.True(t, headersPresent)
		assert.Contains(t, err.Error(), "max-participants header is required")
	})

	t.Run("threshold greater than max - error", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS:    "3",
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "5",
			},
		}

		_, headersPresent, err := bpm.GetCrossValidationParameters()
		require.Error(t, err)
		assert.True(t, headersPresent)
		assert.Contains(t, err.Error(), "cannot be greater than max-participants")
	})

	t.Run("invalid max-participants (not a number) - error", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS:    "invalid",
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "3",
			},
		}

		_, headersPresent, err := bpm.GetCrossValidationParameters()
		require.Error(t, err)
		assert.True(t, headersPresent)
		assert.Contains(t, err.Error(), "invalid cross-validation max-participants")
	})

	t.Run("invalid agreement-threshold (not a number) - error", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS:    "5",
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "invalid",
			},
		}

		_, headersPresent, err := bpm.GetCrossValidationParameters()
		require.Error(t, err)
		assert.True(t, headersPresent)
		assert.Contains(t, err.Error(), "invalid cross-validation agreement-threshold")
	})

	t.Run("max-participants zero - error", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS:    "0",
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "0",
			},
		}

		_, headersPresent, err := bpm.GetCrossValidationParameters()
		require.Error(t, err)
		assert.True(t, headersPresent)
		assert.Contains(t, err.Error(), "must be a positive integer")
	})

	t.Run("negative max-participants - error", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS:    "-1",
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "1",
			},
		}

		_, headersPresent, err := bpm.GetCrossValidationParameters()
		require.Error(t, err)
		assert.True(t, headersPresent)
		assert.Contains(t, err.Error(), "must be a positive integer")
	})

	t.Run("agreement-threshold zero - error", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS:    "5",
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "0",
			},
		}

		_, headersPresent, err := bpm.GetCrossValidationParameters()
		require.Error(t, err)
		assert.True(t, headersPresent)
		assert.Contains(t, err.Error(), "must be a positive integer")
	})

	t.Run("minimum valid values (1, 1) - valid", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS:    "1",
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "1",
			},
		}

		params, headersPresent, err := bpm.GetCrossValidationParameters()
		require.NoError(t, err)
		assert.True(t, headersPresent)
		assert.Equal(t, 1, params.MaxParticipants)
		assert.Equal(t, 1, params.AgreementThreshold)
	})

	t.Run("large valid values - valid", func(t *testing.T) {
		bpm := &BaseProtocolMessage{
			directiveHeaders: map[string]string{
				common.CROSS_VALIDATION_HEADER_MAX_PARTICIPANTS:    "100",
				common.CROSS_VALIDATION_HEADER_AGREEMENT_THRESHOLD: "51",
			},
		}

		params, headersPresent, err := bpm.GetCrossValidationParameters()
		require.NoError(t, err)
		assert.True(t, headersPresent)
		assert.Equal(t, 100, params.MaxParticipants)
		assert.Equal(t, 51, params.AgreementThreshold)
	})
}
