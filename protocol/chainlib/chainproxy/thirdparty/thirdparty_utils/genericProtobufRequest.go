package thirdparty_utils

import "github.com/gogo/protobuf/proto"

type GenericProtobufRequest interface {
	Descriptor() ([]byte, []int)
	Marshal() (dAtA []byte, err error)
	MarshalTo(dAtA []byte) (int, error)
	MarshalToSizedBuffer(dAtA []byte) (int, error)
	ProtoMessage()
	Reset()
	Size() (n int)
	String() string
	Unmarshal(dAtA []byte) error
	XXX_DiscardUnknown()
	XXX_Marshal(b []byte, deterministic bool) ([]byte, error)
	XXX_Merge(src proto.Message)
	XXX_Size() int
	XXX_Unmarshal(b []byte) error
}
