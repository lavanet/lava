package rpcclient

import (
	"github.com/lavanet/lava/v3/utils/sigs"
)

func CreateHashFromParams(params []byte) string {
	return string(sigs.HashMsg(params))
}
