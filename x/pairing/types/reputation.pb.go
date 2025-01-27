// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: lavanet/lava/pairing/reputation.proto

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

// Frac is a fracture struct that helps calculating and updating weighted average on the go
type Frac struct {
	Num   github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,1,opt,name=num,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"num" yaml:"num"`
	Denom github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,2,opt,name=denom,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"denom" yaml:"denom"`
}

func (m *Frac) Reset()         { *m = Frac{} }
func (m *Frac) String() string { return proto.CompactTextString(m) }
func (*Frac) ProtoMessage()    {}
func (*Frac) Descriptor() ([]byte, []int) {
	return fileDescriptor_38105d47def04072, []int{0}
}
func (m *Frac) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Frac) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Frac.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Frac) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Frac.Merge(m, src)
}
func (m *Frac) XXX_Size() int {
	return m.Size()
}
func (m *Frac) XXX_DiscardUnknown() {
	xxx_messageInfo_Frac.DiscardUnknown(m)
}

var xxx_messageInfo_Frac proto.InternalMessageInfo

// QosScore holds the QoS score from a QoS excellence report. The score and its variance are updated over time using a weighted average.
// Currently, the weight is the amount of relays that are associated with the QoS report.
type QosScore struct {
	Score    Frac `protobuf:"bytes,1,opt,name=score,proto3" json:"score"`
	Variance Frac `protobuf:"bytes,2,opt,name=variance,proto3" json:"variance"`
}

func (m *QosScore) Reset()         { *m = QosScore{} }
func (m *QosScore) String() string { return proto.CompactTextString(m) }
func (*QosScore) ProtoMessage()    {}
func (*QosScore) Descriptor() ([]byte, []int) {
	return fileDescriptor_38105d47def04072, []int{1}
}
func (m *QosScore) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QosScore) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QosScore.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QosScore) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QosScore.Merge(m, src)
}
func (m *QosScore) XXX_Size() int {
	return m.Size()
}
func (m *QosScore) XXX_DiscardUnknown() {
	xxx_messageInfo_QosScore.DiscardUnknown(m)
}

var xxx_messageInfo_QosScore proto.InternalMessageInfo

func (m *QosScore) GetScore() Frac {
	if m != nil {
		return m.Score
	}
	return Frac{}
}

func (m *QosScore) GetVariance() Frac {
	if m != nil {
		return m.Variance
	}
	return Frac{}
}

// Reputation keeps the QosScore of a provider for a specific cluster for in the provider's geolocation.
// The store key is provider+chain+cluster.
// The epoch_score is a QosScore object that is aggregated over an epoch. When an epoch ends, the "score" field is updated
// with the epoch_score and the epoch_score is reset.
// The time_last_updated is used to calculate the appropriate time decay upon update.
// The creation_time is used to determine if the variance stabilization period has passed and score can be truncated.
// The stake is used when converting the reputation QoS scores to repuatation pairing score.
type Reputation struct {
	Score           QosScore   `protobuf:"bytes,1,opt,name=score,proto3" json:"score"`
	EpochScore      QosScore   `protobuf:"bytes,2,opt,name=epoch_score,json=epochScore,proto3" json:"epoch_score"`
	TimeLastUpdated int64      `protobuf:"varint,3,opt,name=time_last_updated,json=timeLastUpdated,proto3" json:"time_last_updated,omitempty"`
	CreationTime    int64      `protobuf:"varint,4,opt,name=creation_time,json=creationTime,proto3" json:"creation_time,omitempty"`
	Stake           types.Coin `protobuf:"bytes,5,opt,name=stake,proto3" json:"stake"`
}

func (m *Reputation) Reset()         { *m = Reputation{} }
func (m *Reputation) String() string { return proto.CompactTextString(m) }
func (*Reputation) ProtoMessage()    {}
func (*Reputation) Descriptor() ([]byte, []int) {
	return fileDescriptor_38105d47def04072, []int{2}
}
func (m *Reputation) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Reputation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Reputation.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Reputation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Reputation.Merge(m, src)
}
func (m *Reputation) XXX_Size() int {
	return m.Size()
}
func (m *Reputation) XXX_DiscardUnknown() {
	xxx_messageInfo_Reputation.DiscardUnknown(m)
}

var xxx_messageInfo_Reputation proto.InternalMessageInfo

func (m *Reputation) GetScore() QosScore {
	if m != nil {
		return m.Score
	}
	return QosScore{}
}

func (m *Reputation) GetEpochScore() QosScore {
	if m != nil {
		return m.EpochScore
	}
	return QosScore{}
}

func (m *Reputation) GetTimeLastUpdated() int64 {
	if m != nil {
		return m.TimeLastUpdated
	}
	return 0
}

func (m *Reputation) GetCreationTime() int64 {
	if m != nil {
		return m.CreationTime
	}
	return 0
}

func (m *Reputation) GetStake() types.Coin {
	if m != nil {
		return m.Stake
	}
	return types.Coin{}
}

// ReputationPairingScore holds the reputation pairing score used by the reputation pairing requirement.
// The score is ranged between [0.5-2]. It's kept in the reputations fixation store with a provider+chain+cluster key.
type ReputationPairingScore struct {
	Score github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,1,opt,name=score,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"score" yaml:"score"`
}

func (m *ReputationPairingScore) Reset()         { *m = ReputationPairingScore{} }
func (m *ReputationPairingScore) String() string { return proto.CompactTextString(m) }
func (*ReputationPairingScore) ProtoMessage()    {}
func (*ReputationPairingScore) Descriptor() ([]byte, []int) {
	return fileDescriptor_38105d47def04072, []int{3}
}
func (m *ReputationPairingScore) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ReputationPairingScore) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ReputationPairingScore.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ReputationPairingScore) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReputationPairingScore.Merge(m, src)
}
func (m *ReputationPairingScore) XXX_Size() int {
	return m.Size()
}
func (m *ReputationPairingScore) XXX_DiscardUnknown() {
	xxx_messageInfo_ReputationPairingScore.DiscardUnknown(m)
}

var xxx_messageInfo_ReputationPairingScore proto.InternalMessageInfo

func init() {
	proto.RegisterType((*Frac)(nil), "lavanet.lava.pairing.Frac")
	proto.RegisterType((*QosScore)(nil), "lavanet.lava.pairing.QosScore")
	proto.RegisterType((*Reputation)(nil), "lavanet.lava.pairing.Reputation")
	proto.RegisterType((*ReputationPairingScore)(nil), "lavanet.lava.pairing.ReputationPairingScore")
}

func init() {
	proto.RegisterFile("lavanet/lava/pairing/reputation.proto", fileDescriptor_38105d47def04072)
}

var fileDescriptor_38105d47def04072 = []byte{
	// 466 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x93, 0xcd, 0x6e, 0x13, 0x31,
	0x10, 0xc7, 0xb3, 0xf9, 0x40, 0x65, 0x5a, 0x84, 0x58, 0x55, 0x68, 0xc9, 0xc1, 0xa9, 0x8c, 0x40,
	0x55, 0x25, 0x6c, 0x95, 0xaf, 0x43, 0x55, 0x71, 0x08, 0x85, 0x13, 0x42, 0xb0, 0x84, 0x0b, 0x97,
	0xc8, 0x71, 0xac, 0xd4, 0x6a, 0x6c, 0xaf, 0xd6, 0xde, 0x88, 0xde, 0x78, 0x04, 0x6e, 0xbc, 0x04,
	0x0f, 0xd2, 0x63, 0x8f, 0x88, 0x43, 0x84, 0x92, 0x37, 0xe0, 0x09, 0x90, 0xed, 0x6d, 0x68, 0xa4,
	0x1e, 0xa8, 0x38, 0x8d, 0x65, 0xff, 0xe7, 0xe7, 0xbf, 0xc7, 0x33, 0xf0, 0x60, 0xca, 0x66, 0x4c,
	0x0b, 0x47, 0x7d, 0xa4, 0x05, 0x93, 0xa5, 0xd4, 0x13, 0x5a, 0x8a, 0xa2, 0x72, 0xcc, 0x49, 0xa3,
	0x49, 0x51, 0x1a, 0x67, 0xd2, 0xed, 0x5a, 0x46, 0x7c, 0x24, 0xb5, 0xac, 0xbb, 0x3d, 0x31, 0x13,
	0x13, 0x04, 0xd4, 0xaf, 0xa2, 0xb6, 0x8b, 0xb8, 0xb1, 0xca, 0x58, 0x3a, 0x62, 0x56, 0xd0, 0xd9,
	0xfe, 0x48, 0x38, 0xb6, 0x4f, 0xb9, 0x91, 0x35, 0x0b, 0x7f, 0x4f, 0xa0, 0xfd, 0xba, 0x64, 0x3c,
	0x7d, 0x0b, 0x2d, 0x5d, 0xa9, 0x2c, 0xd9, 0x49, 0x76, 0x6f, 0xf6, 0x0f, 0xcf, 0xe6, 0xbd, 0xc6,
	0xcf, 0x79, 0xef, 0xe1, 0x44, 0xba, 0xe3, 0x6a, 0x44, 0xb8, 0x51, 0xb4, 0x06, 0xc5, 0xf0, 0xc8,
	0x8e, 0x4f, 0xa8, 0x3b, 0x2d, 0x84, 0x25, 0x47, 0x82, 0xff, 0x9e, 0xf7, 0xe0, 0x94, 0xa9, 0xe9,
	0x01, 0xd6, 0x95, 0xc2, 0xb9, 0x07, 0xa5, 0x03, 0xe8, 0x8c, 0x85, 0x36, 0x2a, 0x6b, 0x06, 0xe2,
	0x8b, 0x6b, 0x13, 0xb7, 0x22, 0x31, 0x40, 0x70, 0x1e, 0x61, 0xf8, 0x4b, 0x02, 0x1b, 0xef, 0x8d,
	0xfd, 0xc0, 0x4d, 0x29, 0xd2, 0xe7, 0xd0, 0xb1, 0x7e, 0x11, 0x4c, 0x6f, 0x3e, 0xee, 0x92, 0xab,
	0xea, 0x42, 0xfc, 0xeb, 0xfa, 0x6d, 0x7f, 0x7d, 0x1e, 0xe5, 0xe9, 0x21, 0x6c, 0xcc, 0x58, 0x29,
	0x99, 0xe6, 0x22, 0xb8, 0xfb, 0x97, 0xd4, 0x55, 0x06, 0xfe, 0xd6, 0x04, 0xc8, 0x57, 0x5f, 0x92,
	0x1e, 0xac, 0x9b, 0x40, 0x57, 0x93, 0x2e, 0x3c, 0xaf, 0x1b, 0x79, 0x05, 0x9b, 0xa2, 0x30, 0xfc,
	0x78, 0x18, 0x09, 0xcd, 0x6b, 0x10, 0x20, 0x24, 0xc6, 0x3a, 0xec, 0xc1, 0x1d, 0x27, 0x95, 0x18,
	0x4e, 0x99, 0x75, 0xc3, 0xaa, 0x18, 0x33, 0x27, 0xc6, 0x59, 0x6b, 0x27, 0xd9, 0x6d, 0xe5, 0xb7,
	0xfd, 0xc1, 0x1b, 0x66, 0xdd, 0xc7, 0xb8, 0x9d, 0xde, 0x87, 0x5b, 0xbc, 0x14, 0xc1, 0xfa, 0xd0,
	0x9f, 0x65, 0xed, 0xa0, 0xdb, 0xba, 0xd8, 0x1c, 0x48, 0x25, 0xd2, 0x67, 0xd0, 0xb1, 0x8e, 0x9d,
	0x88, 0xac, 0x13, 0x1c, 0xdd, 0x23, 0xf1, 0x8b, 0x88, 0x6f, 0x22, 0x52, 0x37, 0x11, 0x79, 0x69,
	0xa4, 0x5e, 0x3d, 0xc7, 0xab, 0xb1, 0x86, 0xbb, 0x7f, 0x0b, 0xf3, 0x2e, 0xfa, 0x8e, 0x0e, 0x07,
	0x97, 0x8b, 0xf4, 0x1f, 0xcd, 0x10, 0x20, 0xb8, 0x2e, 0x5f, 0xff, 0xe8, 0x6c, 0x81, 0x92, 0xf3,
	0x05, 0x4a, 0x7e, 0x2d, 0x50, 0xf2, 0x75, 0x89, 0x1a, 0xe7, 0x4b, 0xd4, 0xf8, 0xb1, 0x44, 0x8d,
	0x4f, 0x7b, 0x97, 0xc0, 0x6b, 0x33, 0x35, 0x7b, 0x4a, 0x3f, 0xaf, 0x06, 0x2b, 0x5c, 0x30, 0xba,
	0x11, 0x06, 0xe1, 0xc9, 0x9f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xf8, 0x5a, 0xc9, 0x7a, 0x7d, 0x03,
	0x00, 0x00,
}

func (m *Frac) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Frac) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Frac) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Denom.Size()
		i -= size
		if _, err := m.Denom.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintReputation(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	{
		size := m.Num.Size()
		i -= size
		if _, err := m.Num.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintReputation(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *QosScore) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QosScore) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QosScore) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Variance.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintReputation(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	{
		size, err := m.Score.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintReputation(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *Reputation) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Reputation) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Reputation) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Stake.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintReputation(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x2a
	if m.CreationTime != 0 {
		i = encodeVarintReputation(dAtA, i, uint64(m.CreationTime))
		i--
		dAtA[i] = 0x20
	}
	if m.TimeLastUpdated != 0 {
		i = encodeVarintReputation(dAtA, i, uint64(m.TimeLastUpdated))
		i--
		dAtA[i] = 0x18
	}
	{
		size, err := m.EpochScore.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintReputation(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	{
		size, err := m.Score.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintReputation(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *ReputationPairingScore) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ReputationPairingScore) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ReputationPairingScore) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Score.Size()
		i -= size
		if _, err := m.Score.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintReputation(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintReputation(dAtA []byte, offset int, v uint64) int {
	offset -= sovReputation(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Frac) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Num.Size()
	n += 1 + l + sovReputation(uint64(l))
	l = m.Denom.Size()
	n += 1 + l + sovReputation(uint64(l))
	return n
}

func (m *QosScore) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Score.Size()
	n += 1 + l + sovReputation(uint64(l))
	l = m.Variance.Size()
	n += 1 + l + sovReputation(uint64(l))
	return n
}

func (m *Reputation) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Score.Size()
	n += 1 + l + sovReputation(uint64(l))
	l = m.EpochScore.Size()
	n += 1 + l + sovReputation(uint64(l))
	if m.TimeLastUpdated != 0 {
		n += 1 + sovReputation(uint64(m.TimeLastUpdated))
	}
	if m.CreationTime != 0 {
		n += 1 + sovReputation(uint64(m.CreationTime))
	}
	l = m.Stake.Size()
	n += 1 + l + sovReputation(uint64(l))
	return n
}

func (m *ReputationPairingScore) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Score.Size()
	n += 1 + l + sovReputation(uint64(l))
	return n
}

func sovReputation(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozReputation(x uint64) (n int) {
	return sovReputation(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Frac) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowReputation
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
			return fmt.Errorf("proto: Frac: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Frac: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Num", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReputation
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
				return ErrInvalidLengthReputation
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthReputation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Num.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Denom", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReputation
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
				return ErrInvalidLengthReputation
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthReputation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Denom.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipReputation(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthReputation
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
func (m *QosScore) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowReputation
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
			return fmt.Errorf("proto: QosScore: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QosScore: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Score", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReputation
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
				return ErrInvalidLengthReputation
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthReputation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Score.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Variance", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReputation
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
				return ErrInvalidLengthReputation
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthReputation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Variance.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipReputation(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthReputation
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
func (m *Reputation) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowReputation
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
			return fmt.Errorf("proto: Reputation: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Reputation: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Score", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReputation
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
				return ErrInvalidLengthReputation
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthReputation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Score.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field EpochScore", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReputation
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
				return ErrInvalidLengthReputation
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthReputation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.EpochScore.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TimeLastUpdated", wireType)
			}
			m.TimeLastUpdated = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReputation
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TimeLastUpdated |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreationTime", wireType)
			}
			m.CreationTime = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReputation
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CreationTime |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Stake", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReputation
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
				return ErrInvalidLengthReputation
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthReputation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Stake.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipReputation(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthReputation
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
func (m *ReputationPairingScore) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowReputation
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
			return fmt.Errorf("proto: ReputationPairingScore: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ReputationPairingScore: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Score", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReputation
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
				return ErrInvalidLengthReputation
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthReputation
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Score.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipReputation(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthReputation
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
func skipReputation(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowReputation
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
					return 0, ErrIntOverflowReputation
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
					return 0, ErrIntOverflowReputation
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
				return 0, ErrInvalidLengthReputation
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupReputation
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthReputation
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthReputation        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowReputation          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupReputation = fmt.Errorf("proto: unexpected end of group")
)
