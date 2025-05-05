package chainlib

import (
	"testing"

	"github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/assert"
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
