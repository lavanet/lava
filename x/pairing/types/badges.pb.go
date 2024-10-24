// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: lavanet/lava/pairing/badges.proto

package types

import (
	context "context"
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"
	_ "github.com/lavanet/lava/v4/x/epochstorage/types"
	types "github.com/lavanet/lava/v4/x/spec/types"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	_ "google.golang.org/protobuf/types/known/wrapperspb"
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

type GenerateBadgeRequest struct {
	BadgeAddress string `protobuf:"bytes,1,opt,name=badge_address,json=badgeAddress,proto3" json:"badge_address,omitempty"`
	ProjectId    string `protobuf:"bytes,2,opt,name=project_id,json=projectId,proto3" json:"project_id,omitempty"`
	SpecId       string `protobuf:"bytes,3,opt,name=spec_id,json=specId,proto3" json:"spec_id,omitempty"`
}

func (m *GenerateBadgeRequest) Reset()         { *m = GenerateBadgeRequest{} }
func (m *GenerateBadgeRequest) String() string { return proto.CompactTextString(m) }
func (*GenerateBadgeRequest) ProtoMessage()    {}
func (*GenerateBadgeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_5013dfba46b4caa4, []int{0}
}
func (m *GenerateBadgeRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GenerateBadgeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GenerateBadgeRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GenerateBadgeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenerateBadgeRequest.Merge(m, src)
}
func (m *GenerateBadgeRequest) XXX_Size() int {
	return m.Size()
}
func (m *GenerateBadgeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GenerateBadgeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GenerateBadgeRequest proto.InternalMessageInfo

func (m *GenerateBadgeRequest) GetBadgeAddress() string {
	if m != nil {
		return m.BadgeAddress
	}
	return ""
}

func (m *GenerateBadgeRequest) GetProjectId() string {
	if m != nil {
		return m.ProjectId
	}
	return ""
}

func (m *GenerateBadgeRequest) GetSpecId() string {
	if m != nil {
		return m.SpecId
	}
	return ""
}

type GenerateBadgeResponse struct {
	Badge              *Badge                   `protobuf:"bytes,1,opt,name=badge,proto3" json:"badge,omitempty"`
	GetPairingResponse *QueryGetPairingResponse `protobuf:"bytes,2,opt,name=get_pairing_response,json=getPairingResponse,proto3" json:"get_pairing_response,omitempty"`
	BadgeSignerAddress string                   `protobuf:"bytes,3,opt,name=badge_signer_address,json=badgeSignerAddress,proto3" json:"badge_signer_address,omitempty"`
	Spec               *types.Spec              `protobuf:"bytes,4,opt,name=spec,proto3" json:"spec,omitempty"`
}

func (m *GenerateBadgeResponse) Reset()         { *m = GenerateBadgeResponse{} }
func (m *GenerateBadgeResponse) String() string { return proto.CompactTextString(m) }
func (*GenerateBadgeResponse) ProtoMessage()    {}
func (*GenerateBadgeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_5013dfba46b4caa4, []int{1}
}
func (m *GenerateBadgeResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GenerateBadgeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GenerateBadgeResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GenerateBadgeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenerateBadgeResponse.Merge(m, src)
}
func (m *GenerateBadgeResponse) XXX_Size() int {
	return m.Size()
}
func (m *GenerateBadgeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GenerateBadgeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GenerateBadgeResponse proto.InternalMessageInfo

func (m *GenerateBadgeResponse) GetBadge() *Badge {
	if m != nil {
		return m.Badge
	}
	return nil
}

func (m *GenerateBadgeResponse) GetGetPairingResponse() *QueryGetPairingResponse {
	if m != nil {
		return m.GetPairingResponse
	}
	return nil
}

func (m *GenerateBadgeResponse) GetBadgeSignerAddress() string {
	if m != nil {
		return m.BadgeSignerAddress
	}
	return ""
}

func (m *GenerateBadgeResponse) GetSpec() *types.Spec {
	if m != nil {
		return m.Spec
	}
	return nil
}

func init() {
	proto.RegisterType((*GenerateBadgeRequest)(nil), "lavanet.lava.pairing.GenerateBadgeRequest")
	proto.RegisterType((*GenerateBadgeResponse)(nil), "lavanet.lava.pairing.GenerateBadgeResponse")
}

func init() { proto.RegisterFile("lavanet/lava/pairing/badges.proto", fileDescriptor_5013dfba46b4caa4) }

var fileDescriptor_5013dfba46b4caa4 = []byte{
	// 443 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x52, 0x41, 0x6f, 0xd3, 0x30,
	0x14, 0x6e, 0x46, 0x19, 0x9a, 0xc7, 0x38, 0x58, 0x41, 0x54, 0x85, 0x85, 0x51, 0x2e, 0x68, 0x15,
	0x31, 0x14, 0xfe, 0x00, 0x15, 0xd2, 0xb4, 0x1b, 0x64, 0x37, 0x2e, 0x91, 0x9b, 0x3c, 0xbc, 0x8c,
	0x12, 0x7b, 0xb6, 0x33, 0xa8, 0xc4, 0x2f, 0xe0, 0xc4, 0xcf, 0xda, 0xb1, 0x47, 0x4e, 0x08, 0xb5,
	0x7f, 0x04, 0xf9, 0xd9, 0xad, 0x14, 0x08, 0xd2, 0x2e, 0x89, 0xfd, 0xbe, 0xcf, 0xfe, 0xde, 0xf7,
	0xf9, 0x91, 0x27, 0x73, 0x7e, 0xc5, 0x6b, 0xb0, 0xcc, 0xfd, 0x99, 0xe2, 0x95, 0xae, 0x6a, 0xc1,
	0x66, 0xbc, 0x14, 0x60, 0x52, 0xa5, 0xa5, 0x95, 0x34, 0x0e, 0x94, 0xd4, 0xfd, 0xd3, 0x40, 0x19,
	0x1e, 0x75, 0x1e, 0xd4, 0x30, 0xe7, 0x0b, 0x7f, 0xee, 0x3f, 0x8c, 0xcb, 0x06, 0xf4, 0x86, 0x11,
	0x0b, 0x29, 0x24, 0x2e, 0x99, 0x5b, 0x85, 0x6a, 0x22, 0xa4, 0x14, 0x73, 0x60, 0xb8, 0x9b, 0x35,
	0x1f, 0xd9, 0x17, 0xcd, 0x95, 0x02, 0x1d, 0xfa, 0x19, 0x8e, 0x5b, 0xf7, 0x82, 0x92, 0xc5, 0xb9,
	0xb1, 0x52, 0x73, 0x01, 0xcc, 0x58, 0xfe, 0x09, 0x72, 0xa8, 0xed, 0x56, 0xe2, 0x51, 0x8b, 0x6c,
	0x14, 0x14, 0xf8, 0xf1, 0xe8, 0x68, 0x41, 0xe2, 0x13, 0xa8, 0x41, 0x73, 0x0b, 0x53, 0x67, 0x39,
	0x83, 0xcb, 0x06, 0x8c, 0xa5, 0x4f, 0xc9, 0x01, 0x46, 0x90, 0xf3, 0xb2, 0xd4, 0x60, 0xcc, 0x20,
	0x3a, 0x8a, 0x9e, 0xed, 0x65, 0x77, 0xb1, 0xf8, 0xc6, 0xd7, 0xe8, 0x21, 0x21, 0x4a, 0xcb, 0x0b,
	0x28, 0x6c, 0x5e, 0x95, 0x83, 0x1d, 0x64, 0xec, 0x85, 0xca, 0x69, 0x49, 0x0f, 0xc9, 0x1d, 0xa7,
	0xe4, 0xb0, 0x5b, 0x0e, 0x9b, 0xf6, 0xaf, 0x7f, 0x3d, 0x8e, 0xb2, 0x5d, 0x57, 0x3c, 0x2d, 0x47,
	0xdf, 0x77, 0xc8, 0xfd, 0xbf, 0xb4, 0x8d, 0x92, 0xb5, 0x01, 0xfa, 0x92, 0xdc, 0x46, 0x1d, 0x14,
	0xdd, 0x9f, 0x3c, 0x4c, 0xbb, 0xf2, 0x4f, 0xfd, 0x19, 0xcf, 0xa4, 0x39, 0x89, 0x05, 0xd8, 0x3c,
	0x60, 0xb9, 0x0e, 0x57, 0x61, 0x53, 0xfb, 0x93, 0xe7, 0xdd, 0x37, 0xbc, 0x77, 0x2f, 0x71, 0x02,
	0xf6, 0x9d, 0xdf, 0x6f, 0xf4, 0x33, 0x2a, 0xfe, 0xa9, 0xd1, 0x17, 0x24, 0xf6, 0x81, 0x98, 0x4a,
	0xd4, 0xa0, 0xb7, 0xb9, 0xa0, 0xb3, 0x8c, 0x22, 0x76, 0x86, 0xd0, 0x26, 0x9d, 0x31, 0xe9, 0x3b,
	0xa7, 0x83, 0x3e, 0xb6, 0xf0, 0xa0, 0xdd, 0x02, 0x3e, 0xc1, 0x99, 0x82, 0x22, 0x43, 0xd2, 0xe4,
	0x1b, 0xb9, 0x87, 0x7e, 0x42, 0x20, 0x52, 0xd3, 0x0b, 0x72, 0xd0, 0x4a, 0x87, 0x1e, 0x77, 0x9b,
	0xe8, 0x7a, 0xbe, 0xe1, 0xf8, 0x46, 0x5c, 0x6f, 0x6d, 0xd4, 0x9b, 0xbe, 0xbd, 0x5e, 0x25, 0xd1,
	0x72, 0x95, 0x44, 0xbf, 0x57, 0x49, 0xf4, 0x63, 0x9d, 0xf4, 0x96, 0xeb, 0xa4, 0xf7, 0x73, 0x9d,
	0xf4, 0x3e, 0x1c, 0x8b, 0xca, 0x9e, 0x37, 0xb3, 0xb4, 0x90, 0x9f, 0x59, 0x6b, 0x90, 0xae, 0x5e,
	0xb3, 0xaf, 0xdb, 0x91, 0xb6, 0x0b, 0x05, 0x66, 0xb6, 0x8b, 0x23, 0xf5, 0xea, 0x4f, 0x00, 0x00,
	0x00, 0xff, 0xff, 0x43, 0x04, 0x8b, 0xb8, 0x52, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// BadgeGeneratorClient is the client API for BadgeGenerator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type BadgeGeneratorClient interface {
	GenerateBadge(ctx context.Context, in *GenerateBadgeRequest, opts ...grpc.CallOption) (*GenerateBadgeResponse, error)
}

type badgeGeneratorClient struct {
	cc grpc1.ClientConn
}

func NewBadgeGeneratorClient(cc grpc1.ClientConn) BadgeGeneratorClient {
	return &badgeGeneratorClient{cc}
}

func (c *badgeGeneratorClient) GenerateBadge(ctx context.Context, in *GenerateBadgeRequest, opts ...grpc.CallOption) (*GenerateBadgeResponse, error) {
	out := new(GenerateBadgeResponse)
	err := c.cc.Invoke(ctx, "/lavanet.lava.pairing.BadgeGenerator/GenerateBadge", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BadgeGeneratorServer is the server API for BadgeGenerator service.
type BadgeGeneratorServer interface {
	GenerateBadge(context.Context, *GenerateBadgeRequest) (*GenerateBadgeResponse, error)
}

// UnimplementedBadgeGeneratorServer can be embedded to have forward compatible implementations.
type UnimplementedBadgeGeneratorServer struct {
}

func (*UnimplementedBadgeGeneratorServer) GenerateBadge(ctx context.Context, req *GenerateBadgeRequest) (*GenerateBadgeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenerateBadge not implemented")
}

func RegisterBadgeGeneratorServer(s grpc1.Server, srv BadgeGeneratorServer) {
	s.RegisterService(&_BadgeGenerator_serviceDesc, srv)
}

func _BadgeGenerator_GenerateBadge_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GenerateBadgeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BadgeGeneratorServer).GenerateBadge(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/lavanet.lava.pairing.BadgeGenerator/GenerateBadge",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BadgeGeneratorServer).GenerateBadge(ctx, req.(*GenerateBadgeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _BadgeGenerator_serviceDesc = grpc.ServiceDesc{
	ServiceName: "lavanet.lava.pairing.BadgeGenerator",
	HandlerType: (*BadgeGeneratorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GenerateBadge",
			Handler:    _BadgeGenerator_GenerateBadge_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "lavanet/lava/pairing/badges.proto",
}

func (m *GenerateBadgeRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GenerateBadgeRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GenerateBadgeRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.SpecId) > 0 {
		i -= len(m.SpecId)
		copy(dAtA[i:], m.SpecId)
		i = encodeVarintBadges(dAtA, i, uint64(len(m.SpecId)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.ProjectId) > 0 {
		i -= len(m.ProjectId)
		copy(dAtA[i:], m.ProjectId)
		i = encodeVarintBadges(dAtA, i, uint64(len(m.ProjectId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.BadgeAddress) > 0 {
		i -= len(m.BadgeAddress)
		copy(dAtA[i:], m.BadgeAddress)
		i = encodeVarintBadges(dAtA, i, uint64(len(m.BadgeAddress)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *GenerateBadgeResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GenerateBadgeResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GenerateBadgeResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Spec != nil {
		{
			size, err := m.Spec.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintBadges(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x22
	}
	if len(m.BadgeSignerAddress) > 0 {
		i -= len(m.BadgeSignerAddress)
		copy(dAtA[i:], m.BadgeSignerAddress)
		i = encodeVarintBadges(dAtA, i, uint64(len(m.BadgeSignerAddress)))
		i--
		dAtA[i] = 0x1a
	}
	if m.GetPairingResponse != nil {
		{
			size, err := m.GetPairingResponse.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintBadges(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if m.Badge != nil {
		{
			size, err := m.Badge.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintBadges(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintBadges(dAtA []byte, offset int, v uint64) int {
	offset -= sovBadges(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GenerateBadgeRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.BadgeAddress)
	if l > 0 {
		n += 1 + l + sovBadges(uint64(l))
	}
	l = len(m.ProjectId)
	if l > 0 {
		n += 1 + l + sovBadges(uint64(l))
	}
	l = len(m.SpecId)
	if l > 0 {
		n += 1 + l + sovBadges(uint64(l))
	}
	return n
}

func (m *GenerateBadgeResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Badge != nil {
		l = m.Badge.Size()
		n += 1 + l + sovBadges(uint64(l))
	}
	if m.GetPairingResponse != nil {
		l = m.GetPairingResponse.Size()
		n += 1 + l + sovBadges(uint64(l))
	}
	l = len(m.BadgeSignerAddress)
	if l > 0 {
		n += 1 + l + sovBadges(uint64(l))
	}
	if m.Spec != nil {
		l = m.Spec.Size()
		n += 1 + l + sovBadges(uint64(l))
	}
	return n
}

func sovBadges(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozBadges(x uint64) (n int) {
	return sovBadges(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GenerateBadgeRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBadges
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
			return fmt.Errorf("proto: GenerateBadgeRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GenerateBadgeRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BadgeAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBadges
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
				return ErrInvalidLengthBadges
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthBadges
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BadgeAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProjectId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBadges
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
				return ErrInvalidLengthBadges
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthBadges
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ProjectId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SpecId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBadges
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
				return ErrInvalidLengthBadges
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthBadges
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SpecId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipBadges(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBadges
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
func (m *GenerateBadgeResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBadges
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
			return fmt.Errorf("proto: GenerateBadgeResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GenerateBadgeResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Badge", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBadges
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
				return ErrInvalidLengthBadges
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthBadges
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Badge == nil {
				m.Badge = &Badge{}
			}
			if err := m.Badge.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field GetPairingResponse", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBadges
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
				return ErrInvalidLengthBadges
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthBadges
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.GetPairingResponse == nil {
				m.GetPairingResponse = &QueryGetPairingResponse{}
			}
			if err := m.GetPairingResponse.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BadgeSignerAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBadges
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
				return ErrInvalidLengthBadges
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthBadges
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BadgeSignerAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Spec", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBadges
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
				return ErrInvalidLengthBadges
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthBadges
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Spec == nil {
				m.Spec = &types.Spec{}
			}
			if err := m.Spec.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipBadges(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBadges
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
func skipBadges(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowBadges
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
					return 0, ErrIntOverflowBadges
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
					return 0, ErrIntOverflowBadges
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
				return 0, ErrInvalidLengthBadges
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupBadges
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthBadges
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthBadges        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowBadges          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupBadges = fmt.Errorf("proto: unexpected end of group")
)
