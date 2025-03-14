// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: lavanet/lava/dualstaking/delegator_reward.proto

package types

import (
	fmt "fmt"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type DelegatorReward struct {
	Delegator string                                   `protobuf:"bytes,1,opt,name=delegator,proto3" json:"delegator,omitempty"`
	Provider  string                                   `protobuf:"bytes,2,opt,name=provider,proto3" json:"provider,omitempty"`
	Amount    github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,4,rep,name=amount,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"amount"`
}

func (m *DelegatorReward) Reset()         { *m = DelegatorReward{} }
func (m *DelegatorReward) String() string { return proto.CompactTextString(m) }
func (*DelegatorReward) ProtoMessage()    {}
func (*DelegatorReward) Descriptor() ([]byte, []int) {
	return fileDescriptor_c8b6da054bf40d1f, []int{0}
}
func (m *DelegatorReward) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DelegatorReward) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DelegatorReward.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DelegatorReward) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DelegatorReward.Merge(m, src)
}
func (m *DelegatorReward) XXX_Size() int {
	return m.Size()
}
func (m *DelegatorReward) XXX_DiscardUnknown() {
	xxx_messageInfo_DelegatorReward.DiscardUnknown(m)
}

var xxx_messageInfo_DelegatorReward proto.InternalMessageInfo

func (m *DelegatorReward) GetDelegator() string {
	if m != nil {
		return m.Delegator
	}
	return ""
}

func (m *DelegatorReward) GetProvider() string {
	if m != nil {
		return m.Provider
	}
	return ""
}

func (m *DelegatorReward) GetAmount() github_com_cosmos_cosmos_sdk_types.Coins {
	if m != nil {
		return m.Amount
	}
	return nil
}

func init() {
	proto.RegisterType((*DelegatorReward)(nil), "lavanet.lava.dualstaking.DelegatorReward")
}

func init() {
	proto.RegisterFile("lavanet/lava/dualstaking/delegator_reward.proto", fileDescriptor_c8b6da054bf40d1f)
}

var fileDescriptor_c8b6da054bf40d1f = []byte{
	// 292 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x54, 0x90, 0x31, 0x4e, 0xfb, 0x30,
	0x18, 0xc5, 0xe3, 0x7f, 0xa3, 0xaa, 0xcd, 0x7f, 0x00, 0x45, 0x0c, 0x21, 0x42, 0x6e, 0xc5, 0x94,
	0x05, 0x9b, 0x82, 0xb8, 0x40, 0x61, 0x40, 0x8c, 0x19, 0x59, 0x90, 0x93, 0x58, 0x21, 0x6a, 0xe2,
	0x2f, 0xb2, 0x9d, 0x00, 0xb7, 0xe0, 0x1c, 0x1c, 0x80, 0x33, 0x74, 0xec, 0xc8, 0x04, 0x28, 0xb9,
	0x08, 0x8a, 0x13, 0x4a, 0x3b, 0x3d, 0xdb, 0xdf, 0xf7, 0xde, 0xcf, 0x7a, 0x0e, 0xcd, 0x59, 0xcd,
	0x04, 0xd7, 0x46, 0x69, 0x52, 0xb1, 0x5c, 0x69, 0xb6, 0xca, 0x44, 0x4a, 0x13, 0x9e, 0xf3, 0x94,
	0x69, 0x90, 0x0f, 0x92, 0x3f, 0x31, 0x99, 0x90, 0x52, 0x82, 0x06, 0xd7, 0x1b, 0x0c, 0xa4, 0x53,
	0xb2, 0x63, 0xf0, 0x8f, 0x52, 0x48, 0xc1, 0x2c, 0xd1, 0xee, 0xd4, 0xef, 0xfb, 0x38, 0x06, 0x55,
	0x80, 0xa2, 0x11, 0x53, 0x9c, 0xd6, 0x8b, 0x88, 0x6b, 0xb6, 0xa0, 0x31, 0x64, 0xa2, 0x9f, 0x9f,
	0xbe, 0x23, 0xe7, 0xe0, 0xe6, 0x17, 0x15, 0x1a, 0x92, 0x7b, 0xe2, 0x4c, 0xb7, 0x74, 0x0f, 0xcd,
	0x51, 0x30, 0x0d, 0xff, 0x1e, 0x5c, 0xdf, 0x99, 0x94, 0x12, 0xea, 0x2c, 0xe1, 0xd2, 0xfb, 0x67,
	0x86, 0xdb, 0xbb, 0x1b, 0x3b, 0x63, 0x56, 0x40, 0x25, 0xb4, 0x67, 0xcf, 0x47, 0xc1, 0xff, 0x8b,
	0x63, 0xd2, 0xe3, 0x49, 0x87, 0x27, 0x03, 0x9e, 0x5c, 0x43, 0x26, 0x96, 0xe7, 0xeb, 0xcf, 0x99,
	0xf5, 0xf6, 0x35, 0x0b, 0xd2, 0x4c, 0x3f, 0x56, 0x11, 0x89, 0xa1, 0xa0, 0xc3, 0x5f, 0x7b, 0x39,
	0x53, 0xc9, 0x8a, 0xea, 0x97, 0x92, 0x2b, 0x63, 0x50, 0xe1, 0x10, 0x7d, 0x67, 0x4f, 0x46, 0x87,
	0xf6, 0xf2, 0x76, 0xdd, 0x60, 0xb4, 0x69, 0x30, 0xfa, 0x6e, 0x30, 0x7a, 0x6d, 0xb1, 0xb5, 0x69,
	0xb1, 0xf5, 0xd1, 0x62, 0xeb, 0x9e, 0xec, 0x24, 0xee, 0xd5, 0x5b, 0x5f, 0xd1, 0xe7, 0xbd, 0x8e,
	0x4d, 0x7a, 0x34, 0x36, 0x4d, 0x5c, 0xfe, 0x04, 0x00, 0x00, 0xff, 0xff, 0xa3, 0x54, 0x91, 0xbd,
	0x8c, 0x01, 0x00, 0x00,
}

func (m *DelegatorReward) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DelegatorReward) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DelegatorReward) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Amount) > 0 {
		for iNdEx := len(m.Amount) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Amount[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintDelegatorReward(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.Provider) > 0 {
		i -= len(m.Provider)
		copy(dAtA[i:], m.Provider)
		i = encodeVarintDelegatorReward(dAtA, i, uint64(len(m.Provider)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Delegator) > 0 {
		i -= len(m.Delegator)
		copy(dAtA[i:], m.Delegator)
		i = encodeVarintDelegatorReward(dAtA, i, uint64(len(m.Delegator)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintDelegatorReward(dAtA []byte, offset int, v uint64) int {
	offset -= sovDelegatorReward(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *DelegatorReward) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Delegator)
	if l > 0 {
		n += 1 + l + sovDelegatorReward(uint64(l))
	}
	l = len(m.Provider)
	if l > 0 {
		n += 1 + l + sovDelegatorReward(uint64(l))
	}
	if len(m.Amount) > 0 {
		for _, e := range m.Amount {
			l = e.Size()
			n += 1 + l + sovDelegatorReward(uint64(l))
		}
	}
	return n
}

func sovDelegatorReward(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozDelegatorReward(x uint64) (n int) {
	return sovDelegatorReward(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *DelegatorReward) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDelegatorReward
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DelegatorReward: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DelegatorReward: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Delegator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDelegatorReward
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthDelegatorReward
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthDelegatorReward
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Delegator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Provider", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDelegatorReward
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthDelegatorReward
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthDelegatorReward
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Provider = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDelegatorReward
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthDelegatorReward
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDelegatorReward
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Amount = append(m.Amount, types.Coin{})
			if err := m.Amount[len(m.Amount)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipDelegatorReward(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthDelegatorReward
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipDelegatorReward(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowDelegatorReward
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowDelegatorReward
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowDelegatorReward
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthDelegatorReward
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupDelegatorReward
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthDelegatorReward
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthDelegatorReward        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowDelegatorReward          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupDelegatorReward = fmt.Errorf("proto: unexpected end of group")
)
