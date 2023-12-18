package protocopy

import (
	"github.com/lavanet/lava/utils"
)

type protoTypeOut interface {
	Unmarshal(dAtA []byte) error
}

type protoTypeIn interface {
	Marshal() (dAtA []byte, err error)
}

func DeepCopyProtoObject(protoIn protoTypeIn, protoOut protoTypeOut) error {
	// Marshal input as an intermediate representation
	jsonData, err := protoIn.Marshal()
	if err != nil {
		return utils.LavaFormatError("Failed marshaling DeepCopyProtoObject", err)
	}

	// Unmarshal into output using JSON
	if err := protoOut.Unmarshal(jsonData); err != nil {
		return utils.LavaFormatError("Failed unmarshaling DeepCopyProtoObject", err)
	}
	return nil
}
