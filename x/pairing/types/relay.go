package types

import (
	"encoding/binary"
)

// Metadata represents a key-value pair attached to a relay request or reply.
type Metadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (m *Metadata) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Metadata) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

// RelayPrivateData_TaskId is the oneof wrapper for TaskId inside RelayPrivateData.
type RelayPrivateData_TaskId struct {
	TaskId string
}

// RelayPrivateData_TxId is the oneof wrapper for TxId inside RelayPrivateData.
type RelayPrivateData_TxId struct {
	TxId string
}

// RelayPrivateData contains the private (client-side) relay request fields.
type RelayPrivateData struct {
	ConnectionType string     `json:"connection_type"`
	ApiUrl         string     `json:"api_url"`
	Data           []byte     `json:"data"`
	RequestBlock   int64      `json:"request_block"`
	ApiInterface   string     `json:"api_interface"`
	Salt           []byte     `json:"salt"`
	Metadata       []Metadata `json:"metadata"`
	Addon          string     `json:"addon"`
	Extensions     []string   `json:"extensions"`
	SeenBlock      int64      `json:"seen_block"`
	RequestId      string     `json:"request_id"`

	// Oneof: either XTaskId or XTxId may be set, but not both.
	XTaskId *RelayPrivateData_TaskId `json:"x_task_id,omitempty"`
	XTxId   *RelayPrivateData_TxId   `json:"x_tx_id,omitempty"`
}

func (r *RelayPrivateData) GetConnectionType() string {
	if r != nil {
		return r.ConnectionType
	}
	return ""
}

func (r *RelayPrivateData) GetApiUrl() string {
	if r != nil {
		return r.ApiUrl
	}
	return ""
}

func (r *RelayPrivateData) GetData() []byte {
	if r != nil {
		return r.Data
	}
	return nil
}

func (r *RelayPrivateData) GetRequestBlock() int64 {
	if r != nil {
		return r.RequestBlock
	}
	return 0
}

func (r *RelayPrivateData) GetApiInterface() string {
	if r != nil {
		return r.ApiInterface
	}
	return ""
}

func (r *RelayPrivateData) GetSalt() []byte {
	if r != nil {
		return r.Salt
	}
	return nil
}

func (r *RelayPrivateData) GetMetadata() []Metadata {
	if r != nil {
		return r.Metadata
	}
	return nil
}

func (r *RelayPrivateData) GetAddon() string {
	if r != nil {
		return r.Addon
	}
	return ""
}

func (r *RelayPrivateData) GetExtensions() []string {
	if r != nil {
		return r.Extensions
	}
	return nil
}

func (r *RelayPrivateData) GetSeenBlock() int64 {
	if r != nil {
		return r.SeenBlock
	}
	return 0
}

func (r *RelayPrivateData) GetRequestId() string {
	if r != nil {
		return r.RequestId
	}
	return ""
}

// GetTaskId returns the TaskId from the oneof wrapper, or empty string if unset.
func (r *RelayPrivateData) GetTaskId() string {
	if r != nil && r.XTaskId != nil {
		return r.XTaskId.TaskId
	}
	return ""
}

// GetTxId returns the TxId from the oneof wrapper, or empty string if unset.
func (r *RelayPrivateData) GetTxId() string {
	if r != nil && r.XTxId != nil {
		return r.XTxId.TxId
	}
	return ""
}

// GetContentHashData returns a deterministic byte representation of the relay
// data fields used to compute the session content hash.  The encoding is
// intentionally simple so it can be hashed with sigs.HashMsg.
func (r *RelayPrivateData) GetContentHashData() []byte {
	if r == nil {
		return nil
	}

	var buf []byte

	appendString := func(s string) {
		lenBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(lenBuf, uint32(len(s)))
		buf = append(buf, lenBuf...)
		buf = append(buf, []byte(s)...)
	}
	appendBytes := func(b []byte) {
		lenBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(lenBuf, uint32(len(b)))
		buf = append(buf, lenBuf...)
		buf = append(buf, b...)
	}
	appendInt64 := func(v int64) {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(v))
		buf = append(buf, b...)
	}

	appendString(r.ConnectionType)
	appendString(r.ApiUrl)
	appendBytes(r.Data)
	appendInt64(r.RequestBlock)
	appendString(r.ApiInterface)
	appendBytes(r.Salt)
	appendString(r.Addon)

	for _, ext := range r.Extensions {
		appendString(ext)
	}

	for _, m := range r.Metadata {
		appendString(m.Name)
		appendString(m.Value)
	}

	appendInt64(r.SeenBlock)
	appendString(r.RequestId)
	appendString(r.GetTaskId())
	appendString(r.GetTxId())

	return buf
}

// Marshal serialises RelayPrivateData using JSON for use with protocopy.DeepCopyProtoObject.
func (r *RelayPrivateData) Marshal() ([]byte, error) {
	return jsonMarshal(r)
}

// Unmarshal deserialises RelayPrivateData from JSON.
func (r *RelayPrivateData) Unmarshal(dAtA []byte) error {
	return jsonUnmarshal(dAtA, r)
}

// RelayReply is the response from a provider for a relay request.
type RelayReply struct {
	Data                  []byte     `json:"data"`
	Sig                   []byte     `json:"sig"`
	LatestBlock           int64      `json:"latest_block"`
	FinalizedBlocksHashes []byte     `json:"finalized_blocks_hashes"`
	SigBlocks             []byte     `json:"sig_blocks"`
	Metadata              []Metadata `json:"metadata"`
}

func (r *RelayReply) GetData() []byte {
	if r != nil {
		return r.Data
	}
	return nil
}

func (r *RelayReply) GetSig() []byte {
	if r != nil {
		return r.Sig
	}
	return nil
}

func (r *RelayReply) GetLatestBlock() int64 {
	if r != nil {
		return r.LatestBlock
	}
	return 0
}

func (r *RelayReply) GetFinalizedBlocksHashes() []byte {
	if r != nil {
		return r.FinalizedBlocksHashes
	}
	return nil
}

func (r *RelayReply) GetSigBlocks() []byte {
	if r != nil {
		return r.SigBlocks
	}
	return nil
}

func (r *RelayReply) GetMetadata() []Metadata {
	if r != nil {
		return r.Metadata
	}
	return nil
}

// QualityOfServiceReport holds per-session quality metrics reported to providers.
// Fields are float64 (previously sdk.Dec); values are in [0,1] for availability
// and in seconds for latency and sync.
type QualityOfServiceReport struct {
	Latency      float64 `json:"latency"`
	Availability float64 `json:"availability"`
	Sync         float64 `json:"sync"`
}

func (q *QualityOfServiceReport) GetLatency() float64 {
	if q != nil {
		return q.Latency
	}
	return 0
}

func (q *QualityOfServiceReport) GetAvailability() float64 {
	if q != nil {
		return q.Availability
	}
	return 0
}

func (q *QualityOfServiceReport) GetSync() float64 {
	if q != nil {
		return q.Sync
	}
	return 0
}

// ReportedProvider describes a provider that was reported as unresponsive.
type ReportedProvider struct {
	Address        string `json:"address"`
	Disconnections uint64 `json:"disconnections"`
	Errors         uint64 `json:"errors"`
	TimestampS     int64  `json:"timestamp_s"`
}

func (rp *ReportedProvider) GetAddress() string {
	if rp != nil {
		return rp.Address
	}
	return ""
}

func (rp *ReportedProvider) GetDisconnections() uint64 {
	if rp != nil {
		return rp.Disconnections
	}
	return 0
}

func (rp *ReportedProvider) GetErrors() uint64 {
	if rp != nil {
		return rp.Errors
	}
	return 0
}

func (rp *ReportedProvider) GetTimestampS() int64 {
	if rp != nil {
		return rp.TimestampS
	}
	return 0
}

// Badge is an optional consumer badge attached to a relay session.
type Badge struct {
	CuAllocation uint64 `json:"cu_allocation"`
	Epoch        uint64 `json:"epoch"`
	Address      string `json:"address"`
	LavaChainId  string `json:"lava_chain_id"`
	ProjectSig   []byte `json:"project_sig"`
	VirtualEpoch uint64 `json:"virtual_epoch"`
}

func (b *Badge) GetCuAllocation() uint64 {
	if b != nil {
		return b.CuAllocation
	}
	return 0
}

func (b *Badge) GetEpoch() uint64 {
	if b != nil {
		return b.Epoch
	}
	return 0
}

func (b *Badge) GetAddress() string {
	if b != nil {
		return b.Address
	}
	return ""
}

func (b *Badge) GetLavaChainId() string {
	if b != nil {
		return b.LavaChainId
	}
	return ""
}

func (b *Badge) GetProjectSig() []byte {
	if b != nil {
		return b.ProjectSig
	}
	return nil
}

func (b *Badge) GetVirtualEpoch() uint64 {
	if b != nil {
		return b.VirtualEpoch
	}
	return 0
}

// RelaySession is the on-chain portion of a relay request, signed by the consumer.
type RelaySession struct {
	SpecId                string                  `json:"spec_id"`
	ContentHash           []byte                  `json:"content_hash"`
	SessionId             uint64                  `json:"session_id"`
	CuSum                 uint64                  `json:"cu_sum"`
	Provider              string                  `json:"provider"`
	RelayNum              uint64                  `json:"relay_num"`
	QosReport             *QualityOfServiceReport `json:"qos_report"`
	Epoch                 int64                   `json:"epoch"`
	UnresponsiveProviders []*ReportedProvider     `json:"unresponsive_providers"`
	LavaChainId           string                  `json:"lava_chain_id"`
	Sig                   []byte                  `json:"sig"`
	Badge                 *Badge                  `json:"badge"`
	QosExcellenceReport   *QualityOfServiceReport `json:"qos_excellence_report"`
}

func (rs *RelaySession) GetSpecId() string {
	if rs != nil {
		return rs.SpecId
	}
	return ""
}

func (rs *RelaySession) GetContentHash() []byte {
	if rs != nil {
		return rs.ContentHash
	}
	return nil
}

func (rs *RelaySession) GetSessionId() uint64 {
	if rs != nil {
		return rs.SessionId
	}
	return 0
}

func (rs *RelaySession) GetCuSum() uint64 {
	if rs != nil {
		return rs.CuSum
	}
	return 0
}

func (rs *RelaySession) GetProvider() string {
	if rs != nil {
		return rs.Provider
	}
	return ""
}

func (rs *RelaySession) GetRelayNum() uint64 {
	if rs != nil {
		return rs.RelayNum
	}
	return 0
}

func (rs *RelaySession) GetQosReport() *QualityOfServiceReport {
	if rs != nil {
		return rs.QosReport
	}
	return nil
}

func (rs *RelaySession) GetEpoch() int64 {
	if rs != nil {
		return rs.Epoch
	}
	return 0
}

func (rs *RelaySession) GetUnresponsiveProviders() []*ReportedProvider {
	if rs != nil {
		return rs.UnresponsiveProviders
	}
	return nil
}

func (rs *RelaySession) GetLavaChainId() string {
	if rs != nil {
		return rs.LavaChainId
	}
	return ""
}

func (rs *RelaySession) GetSig() []byte {
	if rs != nil {
		return rs.Sig
	}
	return nil
}

func (rs *RelaySession) GetBadge() *Badge {
	if rs != nil {
		return rs.Badge
	}
	return nil
}

func (rs *RelaySession) GetQosExcellenceReport() *QualityOfServiceReport {
	if rs != nil {
		return rs.QosExcellenceReport
	}
	return nil
}

// GetSignature implements sigs.Signable by returning the session signature.
func (rs RelaySession) GetSignature() []byte {
	return rs.Sig
}

// DataToSign implements sigs.Signable for RelaySession.
func (rs RelaySession) DataToSign() []byte {
	var buf []byte

	appendString := func(s string) {
		lenBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(lenBuf, uint32(len(s)))
		buf = append(buf, lenBuf...)
		buf = append(buf, []byte(s)...)
	}
	appendBytes := func(b []byte) {
		lenBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(lenBuf, uint32(len(b)))
		buf = append(buf, lenBuf...)
		buf = append(buf, b...)
	}
	appendUint64 := func(v uint64) {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, v)
		buf = append(buf, b...)
	}
	appendInt64 := func(v int64) {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(v))
		buf = append(buf, b...)
	}

	appendString(rs.SpecId)
	appendBytes(rs.ContentHash)
	appendUint64(rs.SessionId)
	appendUint64(rs.CuSum)
	appendString(rs.Provider)
	appendUint64(rs.RelayNum)
	appendInt64(rs.Epoch)
	appendString(rs.LavaChainId)

	return buf
}

// HashRounds implements sigs.Signable.
func (rs RelaySession) HashRounds() int {
	return 1
}

// RelayRequest pairs a signed session header with the private relay data.
type RelayRequest struct {
	RelaySession *RelaySession    `json:"relay_session"`
	RelayData    *RelayPrivateData `json:"relay_data"`
}

func (r *RelayRequest) GetRelaySession() *RelaySession {
	if r != nil {
		return r.RelaySession
	}
	return nil
}

func (r *RelayRequest) GetRelayData() *RelayPrivateData {
	if r != nil {
		return r.RelayData
	}
	return nil
}

// RelayExchange pairs a request and reply for signing/verification by sigs.Sign.
type RelayExchange struct {
	Request RelayRequest
	Reply   RelayReply
}

// NewRelayExchange constructs a RelayExchange used for signing relay responses.
func NewRelayExchange(request RelayRequest, reply RelayReply) RelayExchange {
	return RelayExchange{Request: request, Reply: reply}
}

// GetSignature implements sigs.Signable; the signature lives in the reply.
func (re RelayExchange) GetSignature() []byte {
	return re.Reply.Sig
}

// DataToSign implements sigs.Signable; encodes the session and reply fields that
// must be covered by the provider's signature.
func (re RelayExchange) DataToSign() []byte {
	var buf []byte

	appendBytes := func(b []byte) {
		lenBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(lenBuf, uint32(len(b)))
		buf = append(buf, lenBuf...)
		buf = append(buf, b...)
	}
	appendInt64 := func(v int64) {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(v))
		buf = append(buf, b...)
	}

	if re.Request.RelaySession != nil {
		sessionData := re.Request.RelaySession.DataToSign()
		appendBytes(sessionData)
	}
	appendBytes(re.Reply.Data)
	appendInt64(re.Reply.LatestBlock)
	appendBytes(re.Reply.FinalizedBlocksHashes)

	return buf
}

// HashRounds implements sigs.Signable.
func (re RelayExchange) HashRounds() int {
	return 1
}

// ProbeRequest is sent by the consumer to probe a provider before committing to
// a full relay.
type ProbeRequest struct {
	Guid              uint64 `json:"guid"`
	SpecId            string `json:"spec_id"`
	ApiInterface      string `json:"api_interface"`
	WithVerifications bool   `json:"with_verifications"`
}

func (p *ProbeRequest) GetGuid() uint64 {
	if p != nil {
		return p.Guid
	}
	return 0
}

func (p *ProbeRequest) GetSpecId() string {
	if p != nil {
		return p.SpecId
	}
	return ""
}

func (p *ProbeRequest) GetApiInterface() string {
	if p != nil {
		return p.ApiInterface
	}
	return ""
}

func (p *ProbeRequest) GetWithVerifications() bool {
	if p != nil {
		return p.WithVerifications
	}
	return false
}

// Verification holds the result of a single endpoint verification check.
type Verification struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
}

func (v *Verification) GetName() string {
	if v != nil {
		return v.Name
	}
	return ""
}

func (v *Verification) GetPassed() bool {
	if v != nil {
		return v.Passed
	}
	return false
}

// ProbeReply is the response to a ProbeRequest.
type ProbeReply struct {
	Guid                  uint64          `json:"guid"`
	LatestBlock           int64           `json:"latest_block"`
	FinalizedBlocksHashes []byte          `json:"finalized_blocks_hashes"`
	LavaEpoch             uint64          `json:"lava_epoch"`
	LavaLatestBlock       uint64          `json:"lava_latest_block"`
	Verifications         []*Verification `json:"verifications"`
}

func (p *ProbeReply) GetGuid() uint64 {
	if p != nil {
		return p.Guid
	}
	return 0
}

func (p *ProbeReply) GetLatestBlock() int64 {
	if p != nil {
		return p.LatestBlock
	}
	return 0
}

func (p *ProbeReply) GetFinalizedBlocksHashes() []byte {
	if p != nil {
		return p.FinalizedBlocksHashes
	}
	return nil
}

func (p *ProbeReply) GetLavaEpoch() uint64 {
	if p != nil {
		return p.LavaEpoch
	}
	return 0
}

func (p *ProbeReply) GetLavaLatestBlock() uint64 {
	if p != nil {
		return p.LavaLatestBlock
	}
	return 0
}

func (p *ProbeReply) GetVerifications() []*Verification {
	if p != nil {
		return p.Verifications
	}
	return nil
}
