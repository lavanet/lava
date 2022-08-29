package utils

import (
	"encoding/binary"
	"fmt"

	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

func Serialize(data any) []byte {
	switch castedData := data.(type) {
	case uint64:
		res := make([]byte, 8)
		binary.LittleEndian.PutUint64(res, castedData)
		return res
	case pairingtypes.StakeToMaxCUList:
		raw, err := castedData.Marshal()
		if err != nil {
			break
		}
		return raw
	}
	panic(fmt.Sprintf("Lava can't Serialize typetype %T!\n", data))
}

func Deserialize(raw []byte, data any) {
	switch casted := data.(type) {
	case *uint64:
		*casted = binary.LittleEndian.Uint64(raw)
		return
	case *pairingtypes.StakeToMaxCUList:
		err := casted.Unmarshal(raw)
		if err != nil {
			break
		}
		return
	}
	panic(fmt.Sprintf("Lava can't Serialize typetype %T!\n", data))
}
