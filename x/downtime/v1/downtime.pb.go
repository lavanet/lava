// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: lavanet/lava/downtime/v1/downtime.proto

package v1

import (
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	github_com_cosmos_gogoproto_types "github.com/cosmos/gogoproto/types"
	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	io "io"
	math "math"
	math_bits "math/bits"
	time "time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Params defines the parameters of the downtime module.
type Params struct {
	// downtime_duration defines the minimum time elapsed between blocks
	// that we consider the chain to be down.
	DowntimeDuration time.Duration `protobuf:"bytes,1,opt,name=downtime_duration,json=downtimeDuration,proto3,stdduration" json:"downtime_duration"`
	// epoch_duration defines an estimation of the time elapsed between epochs
	EpochDuration time.Duration `protobuf:"bytes,2,opt,name=epoch_duration,json=epochDuration,proto3,stdduration" json:"epoch_duration"`
}

func (m *Params) Reset()         { *m = Params{} }
func (m *Params) String() string { return proto.CompactTextString(m) }
func (*Params) ProtoMessage()    {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_17cbf2f7c6c4bd94, []int{0}
}
func (m *Params) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Params) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Params) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}
func (m *Params) XXX_Size() int {
	return m.Size()
}
func (m *Params) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func (m *Params) GetDowntimeDuration() time.Duration {
	if m != nil {
		return m.DowntimeDuration
	}
	return 0
}

func (m *Params) GetEpochDuration() time.Duration {
	if m != nil {
		return m.EpochDuration
	}
	return 0
}

// Downtime defines a single downtime record.
type Downtime struct {
	// block defines the block that took time to produce.
	Block uint64 `protobuf:"varint,1,opt,name=block,proto3" json:"block,omitempty"`
	// duration defines the time elapsed between the previous block and this one.
	// this defines the effective downtime duration.
	Duration time.Duration `protobuf:"bytes,2,opt,name=duration,proto3,stdduration" json:"duration"`
}

func (m *Downtime) Reset()         { *m = Downtime{} }
func (m *Downtime) String() string { return proto.CompactTextString(m) }
func (*Downtime) ProtoMessage()    {}
func (*Downtime) Descriptor() ([]byte, []int) {
	return fileDescriptor_17cbf2f7c6c4bd94, []int{1}
}
func (m *Downtime) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Downtime) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Downtime.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Downtime) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Downtime.Merge(m, src)
}
func (m *Downtime) XXX_Size() int {
	return m.Size()
}
func (m *Downtime) XXX_DiscardUnknown() {
	xxx_messageInfo_Downtime.DiscardUnknown(m)
}

var xxx_messageInfo_Downtime proto.InternalMessageInfo

func (m *Downtime) GetBlock() uint64 {
	if m != nil {
		return m.Block
	}
	return 0
}

func (m *Downtime) GetDuration() time.Duration {
	if m != nil {
		return m.Duration
	}
	return 0
}

func init() {
	proto.RegisterType((*Params)(nil), "lavanet.lava.downtime.v1.Params")
	proto.RegisterType((*Downtime)(nil), "lavanet.lava.downtime.v1.Downtime")
}

func init() {
	proto.RegisterFile("lavanet/lava/downtime/v1/downtime.proto", fileDescriptor_17cbf2f7c6c4bd94)
}

var fileDescriptor_17cbf2f7c6c4bd94 = []byte{
	// 276 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0xcf, 0x49, 0x2c, 0x4b,
	0xcc, 0x4b, 0x2d, 0xd1, 0x07, 0xd1, 0xfa, 0x29, 0xf9, 0xe5, 0x79, 0x25, 0x99, 0xb9, 0xa9, 0xfa,
	0x65, 0x86, 0x70, 0xb6, 0x5e, 0x41, 0x51, 0x7e, 0x49, 0xbe, 0x90, 0x04, 0x54, 0xa1, 0x1e, 0x88,
	0xd6, 0x83, 0x4b, 0x96, 0x19, 0x4a, 0xc9, 0xa5, 0xe7, 0xe7, 0xa7, 0xe7, 0xa4, 0xea, 0x83, 0xd5,
	0x25, 0x95, 0xa6, 0xe9, 0xa7, 0x94, 0x16, 0x25, 0x96, 0x64, 0xe6, 0xe7, 0x41, 0x74, 0x4a, 0xc9,
	0xa3, 0xcb, 0x83, 0x34, 0x16, 0x97, 0x24, 0xe6, 0x16, 0x40, 0x15, 0x88, 0xa4, 0xe7, 0xa7, 0xe7,
	0x83, 0x99, 0xfa, 0x20, 0x16, 0x44, 0x54, 0x69, 0x19, 0x23, 0x17, 0x5b, 0x40, 0x62, 0x51, 0x62,
	0x6e, 0xb1, 0x50, 0x00, 0x97, 0x20, 0xcc, 0xc2, 0x78, 0x98, 0xe1, 0x12, 0x8c, 0x0a, 0x8c, 0x1a,
	0xdc, 0x46, 0x92, 0x7a, 0x10, 0xd3, 0xf5, 0x60, 0xa6, 0xeb, 0xb9, 0x40, 0x15, 0x38, 0x71, 0x9c,
	0xb8, 0x27, 0xcf, 0x30, 0xe3, 0xbe, 0x3c, 0x63, 0x90, 0x00, 0x4c, 0x37, 0x4c, 0x4e, 0xc8, 0x8b,
	0x8b, 0x2f, 0xb5, 0x20, 0x3f, 0x39, 0x03, 0x61, 0x1c, 0x13, 0xf1, 0xc6, 0xf1, 0x82, 0xb5, 0xc2,
	0x24, 0x94, 0x12, 0xb9, 0x38, 0x5c, 0xa0, 0xe6, 0x0b, 0x89, 0x70, 0xb1, 0x26, 0xe5, 0xe4, 0x27,
	0x67, 0x83, 0x5d, 0xc7, 0x12, 0x04, 0xe1, 0x08, 0xd9, 0x73, 0x71, 0x90, 0x63, 0x0f, 0x5c, 0x93,
	0x93, 0xd3, 0x89, 0x47, 0x72, 0x8c, 0x17, 0x1e, 0xc9, 0x31, 0x3e, 0x78, 0x24, 0xc7, 0x38, 0xe1,
	0xb1, 0x1c, 0xc3, 0x85, 0xc7, 0x72, 0x0c, 0x37, 0x1e, 0xcb, 0x31, 0x44, 0x69, 0xa4, 0x67, 0x96,
	0x64, 0x94, 0x26, 0xe9, 0x25, 0xe7, 0xe7, 0xea, 0xa3, 0x44, 0x65, 0x99, 0xa9, 0x7e, 0x05, 0x72,
	0x7c, 0x26, 0xb1, 0x81, 0xad, 0x32, 0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0xb4, 0x97, 0x7d, 0xea,
	0xf2, 0x01, 0x00, 0x00,
}

func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n1, err1 := github_com_cosmos_gogoproto_types.StdDurationMarshalTo(m.EpochDuration, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.EpochDuration):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintDowntime(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x12
	n2, err2 := github_com_cosmos_gogoproto_types.StdDurationMarshalTo(m.DowntimeDuration, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.DowntimeDuration):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintDowntime(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *Downtime) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Downtime) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Downtime) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n3, err3 := github_com_cosmos_gogoproto_types.StdDurationMarshalTo(m.Duration, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.Duration):])
	if err3 != nil {
		return 0, err3
	}
	i -= n3
	i = encodeVarintDowntime(dAtA, i, uint64(n3))
	i--
	dAtA[i] = 0x12
	if m.Block != 0 {
		i = encodeVarintDowntime(dAtA, i, uint64(m.Block))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintDowntime(dAtA []byte, offset int, v uint64) int {
	offset -= sovDowntime(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.DowntimeDuration)
	n += 1 + l + sovDowntime(uint64(l))
	l = github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.EpochDuration)
	n += 1 + l + sovDowntime(uint64(l))
	return n
}

func (m *Downtime) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Block != 0 {
		n += 1 + sovDowntime(uint64(m.Block))
	}
	l = github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.Duration)
	n += 1 + l + sovDowntime(uint64(l))
	return n
}

func sovDowntime(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozDowntime(x uint64) (n int) {
	return sovDowntime(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDowntime
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
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DowntimeDuration", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDowntime
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
				return ErrInvalidLengthDowntime
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDowntime
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdDurationUnmarshal(&m.DowntimeDuration, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field EpochDuration", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDowntime
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
				return ErrInvalidLengthDowntime
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDowntime
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdDurationUnmarshal(&m.EpochDuration, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipDowntime(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthDowntime
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
func (m *Downtime) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowDowntime
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
			return fmt.Errorf("proto: Downtime: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Downtime: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Block", wireType)
			}
			m.Block = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDowntime
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Block |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Duration", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowDowntime
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
				return ErrInvalidLengthDowntime
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthDowntime
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdDurationUnmarshal(&m.Duration, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipDowntime(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthDowntime
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
func skipDowntime(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowDowntime
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
					return 0, ErrIntOverflowDowntime
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
					return 0, ErrIntOverflowDowntime
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
				return 0, ErrInvalidLengthDowntime
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupDowntime
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthDowntime
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthDowntime        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowDowntime          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupDowntime = fmt.Errorf("proto: unexpected end of group")
)
