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
	// panic:ok: validates that the data is of known type; would fail
	// on start when all parameters are read in.
	panic(fmt.Sprintf("unable to Serialize type %T", data))
}

func Deserialize(raw []byte, data any) {
	switch casted := data.(type) {
	case *uint64:
		*casted = binary.LittleEndian.Uint64(raw)
		return
	}
	// panic:ok: validates that the data is of known type; would fail
	// on start when all parameters are read in.
	panic(fmt.Sprintf("unable to DeSerialize type %T", data))
}
