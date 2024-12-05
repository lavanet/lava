package rpcclient

import (
	"github.com/lavanet/lava/v4/utils/sigs"
)

func CreateHashFromParams(params []byte) string {
	return string(sigs.HashMsg(params))
}
