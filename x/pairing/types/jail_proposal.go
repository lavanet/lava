package types

import (
	"fmt"
	io "io"
	math_bits "math/bits"
	math "math"

	proto "github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// Ensure JailProposal implements v1beta1.Content
var _ v1beta1.Content = &JailProposal{}

// ---- JailProposal ----

type JailProposal struct {
	Title         string             `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	Description   string             `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	ProvidersInfo []ProviderJailInfo `protobuf:"bytes,3,rep,name=providers_info,json=providersInfo,proto3" json:"providers_info"`
}

func (m *JailProposal) Reset()         { *m = JailProposal{} }
func (m *JailProposal) String() string { return proto.CompactTextString(m) }
func (*JailProposal) ProtoMessage()    {}

func (m *JailProposal) XXX_Unmarshal(b []byte) error { return m.Unmarshal(b) }
func (m *JailProposal) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_JailProposal.Marshal(b, m, deterministic)
	}
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *JailProposal) XXX_Merge(src proto.Message) {
	xxx_messageInfo_JailProposal.Merge(m, src)
}
func (m *JailProposal) XXX_Size() int          { return m.Size() }
func (m *JailProposal) XXX_DiscardUnknown()    { xxx_messageInfo_JailProposal.DiscardUnknown(m) }

var xxx_messageInfo_JailProposal proto.InternalMessageInfo

// v1beta1.Content interface
func (p *JailProposal) ProposalRoute() string { return RouterKey }
func (p *JailProposal) ProposalType() string  { return ProposalTypeJail }
func (p *JailProposal) GetTitle() string      { return p.Title }
func (p *JailProposal) GetDescription() string { return p.Description }
func (p *JailProposal) ValidateBasic() error {
	if len(p.ProvidersInfo) == 0 {
		return fmt.Errorf("jail proposal must include at least one provider")
	}
	for _, info := range p.ProvidersInfo {
		if info.Provider == "" {
			return fmt.Errorf("jail proposal provider address cannot be empty")
		}
		if info.ChainId == "" {
			return fmt.Errorf("jail proposal chain_id cannot be empty")
		}
	}
	return nil
}

// ---- ProviderJailInfo ----

type ProviderJailInfo struct {
	Provider    string `protobuf:"bytes,1,opt,name=provider,proto3" json:"provider,omitempty"`
	ChainId     string `protobuf:"bytes,2,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	Reason      string `protobuf:"bytes,3,opt,name=reason,proto3" json:"reason,omitempty"`
	// Unix timestamp (seconds). 0 means permanent jail (stored as math.MaxInt64).
	JailEndTime int64  `protobuf:"varint,4,opt,name=jail_end_time,json=jailEndTime,proto3" json:"jail_end_time,omitempty"`
}

func (m *ProviderJailInfo) Reset()         { *m = ProviderJailInfo{} }
func (m *ProviderJailInfo) String() string { return proto.CompactTextString(m) }
func (*ProviderJailInfo) ProtoMessage()    {}

func (m *ProviderJailInfo) XXX_Unmarshal(b []byte) error { return m.Unmarshal(b) }
func (m *ProviderJailInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ProviderJailInfo.Marshal(b, m, deterministic)
	}
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *ProviderJailInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProviderJailInfo.Merge(m, src)
}
func (m *ProviderJailInfo) XXX_Size() int       { return m.Size() }
func (m *ProviderJailInfo) XXX_DiscardUnknown() { xxx_messageInfo_ProviderJailInfo.DiscardUnknown(m) }

var xxx_messageInfo_ProviderJailInfo proto.InternalMessageInfo

func (m *ProviderJailInfo) GetProvider() string {
	if m != nil {
		return m.Provider
	}
	return ""
}

func (m *ProviderJailInfo) GetChainId() string {
	if m != nil {
		return m.ChainId
	}
	return ""
}

func (m *ProviderJailInfo) GetReason() string {
	if m != nil {
		return m.Reason
	}
	return ""
}

func (m *ProviderJailInfo) GetJailEndTime() int64 {
	if m != nil {
		return m.JailEndTime
	}
	return 0
}

// ---- UnjailProposal ----

// Ensure UnjailProposal implements v1beta1.Content
var _ v1beta1.Content = &UnjailProposal{}

type UnjailProposal struct {
	Title         string             `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	Description   string             `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	ProvidersInfo []ProviderJailInfo `protobuf:"bytes,3,rep,name=providers_info,json=providersInfo,proto3" json:"providers_info"`
}

func (m *UnjailProposal) Reset()         { *m = UnjailProposal{} }
func (m *UnjailProposal) String() string { return proto.CompactTextString(m) }
func (*UnjailProposal) ProtoMessage()    {}

func (m *UnjailProposal) XXX_Unmarshal(b []byte) error { return m.Unmarshal(b) }
func (m *UnjailProposal) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_UnjailProposal.Marshal(b, m, deterministic)
	}
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *UnjailProposal) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UnjailProposal.Merge(m, src)
}
func (m *UnjailProposal) XXX_Size() int          { return m.Size() }
func (m *UnjailProposal) XXX_DiscardUnknown()    { xxx_messageInfo_UnjailProposal.DiscardUnknown(m) }

var xxx_messageInfo_UnjailProposal proto.InternalMessageInfo

func (p *UnjailProposal) ProposalRoute() string  { return RouterKey }
func (p *UnjailProposal) ProposalType() string   { return ProposalTypeUnjail }
func (p *UnjailProposal) GetTitle() string       { return p.Title }
func (p *UnjailProposal) GetDescription() string { return p.Description }
func (p *UnjailProposal) ValidateBasic() error {
	if len(p.ProvidersInfo) == 0 {
		return fmt.Errorf("unjail proposal must include at least one provider")
	}
	for _, info := range p.ProvidersInfo {
		if info.Provider == "" {
			return fmt.Errorf("unjail proposal provider address cannot be empty")
		}
		if info.ChainId == "" {
			return fmt.Errorf("unjail proposal chain_id cannot be empty")
		}
	}
	return nil
}

func (m *UnjailProposal) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *UnjailProposal) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *UnjailProposal) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ProvidersInfo) > 0 {
		for iNdEx := len(m.ProvidersInfo) - 1; iNdEx >= 0; iNdEx-- {
			size, err := m.ProvidersInfo[iNdEx].MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintJailProposal(dAtA, i, uint64(size))
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.Description) > 0 {
		i -= len(m.Description)
		copy(dAtA[i:], m.Description)
		i = encodeVarintJailProposal(dAtA, i, uint64(len(m.Description)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Title) > 0 {
		i -= len(m.Title)
		copy(dAtA[i:], m.Title)
		i = encodeVarintJailProposal(dAtA, i, uint64(len(m.Title)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *UnjailProposal) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Title)
	if l > 0 {
		n += 1 + l + sovJailProposal(uint64(l))
	}
	l = len(m.Description)
	if l > 0 {
		n += 1 + l + sovJailProposal(uint64(l))
	}
	for _, e := range m.ProvidersInfo {
		l = e.Size()
		n += 1 + l + sovJailProposal(uint64(l))
	}
	return n
}

func (m *UnjailProposal) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowJailProposal
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
			return fmt.Errorf("proto: UnjailProposal: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: UnjailProposal: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowJailProposal
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
				return ErrInvalidLengthJailProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 || postIndex > l {
				return ErrInvalidLengthJailProposal
			}
			m.Title = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowJailProposal
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
				return ErrInvalidLengthJailProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 || postIndex > l {
				return ErrInvalidLengthJailProposal
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProvidersInfo", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowJailProposal
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
				return ErrInvalidLengthJailProposal
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 || postIndex > l {
				return ErrInvalidLengthJailProposal
			}
			m.ProvidersInfo = append(m.ProvidersInfo, ProviderJailInfo{})
			if err := m.ProvidersInfo[len(m.ProvidersInfo)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipJailProposal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 || (iNdEx+skippy) > l {
				return ErrInvalidLengthJailProposal
			}
			iNdEx += skippy
		}
	}
	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func init() {
	proto.RegisterType((*JailProposal)(nil), "lavanet.lava.pairing.JailProposal")
	proto.RegisterType((*ProviderJailInfo)(nil), "lavanet.lava.pairing.ProviderJailInfo")
	proto.RegisterType((*UnjailProposal)(nil), "lavanet.lava.pairing.UnjailProposal")
}

// ---- Marshal / Unmarshal for JailProposal ----

func (m *JailProposal) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *JailProposal) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *JailProposal) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ProvidersInfo) > 0 {
		for iNdEx := len(m.ProvidersInfo) - 1; iNdEx >= 0; iNdEx-- {
			size, err := m.ProvidersInfo[iNdEx].MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintJailProposal(dAtA, i, uint64(size))
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.Description) > 0 {
		i -= len(m.Description)
		copy(dAtA[i:], m.Description)
		i = encodeVarintJailProposal(dAtA, i, uint64(len(m.Description)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Title) > 0 {
		i -= len(m.Title)
		copy(dAtA[i:], m.Title)
		i = encodeVarintJailProposal(dAtA, i, uint64(len(m.Title)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *JailProposal) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Title)
	if l > 0 {
		n += 1 + l + sovJailProposal(uint64(l))
	}
	l = len(m.Description)
	if l > 0 {
		n += 1 + l + sovJailProposal(uint64(l))
	}
	for _, e := range m.ProvidersInfo {
		l = e.Size()
		n += 1 + l + sovJailProposal(uint64(l))
	}
	return n
}

func (m *JailProposal) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowJailProposal
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
			return fmt.Errorf("proto: JailProposal: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: JailProposal: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1: // Title
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowJailProposal
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
				return ErrInvalidLengthJailProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 || postIndex > l {
				return ErrInvalidLengthJailProposal
			}
			m.Title = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2: // Description
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowJailProposal
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
				return ErrInvalidLengthJailProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 || postIndex > l {
				return ErrInvalidLengthJailProposal
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3: // ProvidersInfo
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProvidersInfo", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowJailProposal
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
				return ErrInvalidLengthJailProposal
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 || postIndex > l {
				return ErrInvalidLengthJailProposal
			}
			m.ProvidersInfo = append(m.ProvidersInfo, ProviderJailInfo{})
			if err := m.ProvidersInfo[len(m.ProvidersInfo)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipJailProposal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 || (iNdEx+skippy) > l {
				return ErrInvalidLengthJailProposal
			}
			iNdEx += skippy
		}
	}
	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}

// ---- Marshal / Unmarshal for ProviderJailInfo ----

func (m *ProviderJailInfo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ProviderJailInfo) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ProviderJailInfo) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.JailEndTime != 0 {
		i = encodeVarintJailProposal(dAtA, i, uint64(m.JailEndTime))
		i--
		dAtA[i] = 0x20
	}
	if len(m.Reason) > 0 {
		i -= len(m.Reason)
		copy(dAtA[i:], m.Reason)
		i = encodeVarintJailProposal(dAtA, i, uint64(len(m.Reason)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.ChainId) > 0 {
		i -= len(m.ChainId)
		copy(dAtA[i:], m.ChainId)
		i = encodeVarintJailProposal(dAtA, i, uint64(len(m.ChainId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Provider) > 0 {
		i -= len(m.Provider)
		copy(dAtA[i:], m.Provider)
		i = encodeVarintJailProposal(dAtA, i, uint64(len(m.Provider)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ProviderJailInfo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Provider)
	if l > 0 {
		n += 1 + l + sovJailProposal(uint64(l))
	}
	l = len(m.ChainId)
	if l > 0 {
		n += 1 + l + sovJailProposal(uint64(l))
	}
	l = len(m.Reason)
	if l > 0 {
		n += 1 + l + sovJailProposal(uint64(l))
	}
	if m.JailEndTime != 0 {
		n += 1 + sovJailProposal(uint64(m.JailEndTime))
	}
	return n
}

func (m *ProviderJailInfo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowJailProposal
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
			return fmt.Errorf("proto: ProviderJailInfo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ProviderJailInfo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1: // Provider
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Provider", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowJailProposal
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
				return ErrInvalidLengthJailProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 || postIndex > l {
				return ErrInvalidLengthJailProposal
			}
			m.Provider = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2: // ChainId
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChainId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowJailProposal
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
				return ErrInvalidLengthJailProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 || postIndex > l {
				return ErrInvalidLengthJailProposal
			}
			m.ChainId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3: // Reason
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Reason", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowJailProposal
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
				return ErrInvalidLengthJailProposal
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 || postIndex > l {
				return ErrInvalidLengthJailProposal
			}
			m.Reason = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4: // JailEndTime
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field JailEndTime", wireType)
			}
			m.JailEndTime = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowJailProposal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.JailEndTime |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipJailProposal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 || (iNdEx+skippy) > l {
				return ErrInvalidLengthJailProposal
			}
			iNdEx += skippy
		}
	}
	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}

// ---- helpers ----

func encodeVarintJailProposal(dAtA []byte, offset int, v uint64) int {
	offset -= sovJailProposal(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}

func sovJailProposal(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}

func skipJailProposal(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowJailProposal
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
					return 0, ErrIntOverflowJailProposal
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
					return 0, ErrIntOverflowJailProposal
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
				return 0, ErrInvalidLengthJailProposal
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupJailProposal
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthJailProposal
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthJailProposal        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowJailProposal          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupJailProposal = fmt.Errorf("proto: unexpected end of group")
)
