package utils

import (
	"encoding/binary"
	"fmt"
)

func Serialize(data any) []byte {
	switch castedData := data.(type) {
	case uint64:
		res := make([]byte, 8)
		binary.LittleEndian.PutUint64(res, castedData)
		return res
	}
	panic(fmt.Sprintf("Lava can't Serialize typetype %T!\n", data))
}

func Deserialize(raw []byte, data any) {
	switch casted := data.(type) {
	case *uint64:
		*casted = binary.LittleEndian.Uint64(raw)
		return
	}
	panic(fmt.Sprintf("Lava can't DeSerialize typetype %T!\n", data))
}
