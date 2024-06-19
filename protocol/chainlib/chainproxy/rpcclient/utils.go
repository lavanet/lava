package rpcclient

import (
	"github.com/lavanet/lava/utils/sigs"
)

func CreateHashFromParams(params []byte) string {
	return string(sigs.HashMsg(params))
}
