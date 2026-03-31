package relay

import "encoding/json"

// Proto compatibility methods for gRPC wire format.
//
// gRPC's default codec (google.golang.org/grpc/encoding/proto) checks whether
// a message implements protoadapt.MessageV1 (Reset, String, ProtoMessage) and
// then delegates marshaling to the legacy Marshal/Unmarshal methods when
// present.  This lets us use JSON as the wire format for all smart-router
// gRPC calls where both client and server are under our control.
//
// Types that already have Marshal/Unmarshal (RelayRequest, RelayReply,
// RelayPrivateData) only gain the three identity methods here.

// ---------------------------------------------------------------------------
// ProbeRequest
// ---------------------------------------------------------------------------

func (m *ProbeRequest) Reset()        { *m = ProbeRequest{} }
func (m *ProbeRequest) ProtoMessage() {}
func (m *ProbeRequest) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *ProbeRequest) Marshal() ([]byte, error) { return json.Marshal(m) }
func (m *ProbeRequest) Unmarshal(b []byte) error  { return json.Unmarshal(b, m) }

// ---------------------------------------------------------------------------
// ProbeReply
// ---------------------------------------------------------------------------

func (m *ProbeReply) Reset()        { *m = ProbeReply{} }
func (m *ProbeReply) ProtoMessage() {}
func (m *ProbeReply) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *ProbeReply) Marshal() ([]byte, error) { return json.Marshal(m) }
func (m *ProbeReply) Unmarshal(b []byte) error  { return json.Unmarshal(b, m) }

// ---------------------------------------------------------------------------
// RelayRequest — Marshal/Unmarshal already defined in relay.go; add identity
// ---------------------------------------------------------------------------

func (m *RelayRequest) Reset()        { *m = RelayRequest{} }
func (m *RelayRequest) ProtoMessage() {}
func (m *RelayRequest) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

// ---------------------------------------------------------------------------
// RelayReply — Marshal/Unmarshal already defined in relay.go; add identity
// ---------------------------------------------------------------------------

func (m *RelayReply) Reset()        { *m = RelayReply{} }
func (m *RelayReply) ProtoMessage() {}
func (m *RelayReply) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

// ---------------------------------------------------------------------------
// RelayCacheGet
// ---------------------------------------------------------------------------

func (m *RelayCacheGet) Reset()        { *m = RelayCacheGet{} }
func (m *RelayCacheGet) ProtoMessage() {}
func (m *RelayCacheGet) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *RelayCacheGet) Marshal() ([]byte, error) { return json.Marshal(m) }
func (m *RelayCacheGet) Unmarshal(b []byte) error  { return json.Unmarshal(b, m) }

// ---------------------------------------------------------------------------
// CacheRelayReply
// ---------------------------------------------------------------------------

func (m *CacheRelayReply) Reset()        { *m = CacheRelayReply{} }
func (m *CacheRelayReply) ProtoMessage() {}
func (m *CacheRelayReply) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *CacheRelayReply) Marshal() ([]byte, error) { return json.Marshal(m) }
func (m *CacheRelayReply) Unmarshal(b []byte) error  { return json.Unmarshal(b, m) }

// ---------------------------------------------------------------------------
// RelayCacheSet
// ---------------------------------------------------------------------------

func (m *RelayCacheSet) Reset()        { *m = RelayCacheSet{} }
func (m *RelayCacheSet) ProtoMessage() {}
func (m *RelayCacheSet) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *RelayCacheSet) Marshal() ([]byte, error) { return json.Marshal(m) }
func (m *RelayCacheSet) Unmarshal(b []byte) error  { return json.Unmarshal(b, m) }

// ---------------------------------------------------------------------------
// CacheUsage
// ---------------------------------------------------------------------------

func (m *CacheUsage) Reset()        { *m = CacheUsage{} }
func (m *CacheUsage) ProtoMessage() {}
func (m *CacheUsage) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *CacheUsage) Marshal() ([]byte, error) { return json.Marshal(m) }
func (m *CacheUsage) Unmarshal(b []byte) error  { return json.Unmarshal(b, m) }
