package chaintracker

import "encoding/json"

// LatestBlockData is the request for GetLatestBlockData.
type LatestBlockData struct {
	FromBlock     int64 `json:"fromBlock,omitempty"`
	ToBlock       int64 `json:"toBlock,omitempty"`
	SpecificBlock int64 `json:"specificBlock,omitempty"`
}

func (m *LatestBlockData) GetFromBlock() int64 {
	if m != nil {
		return m.FromBlock
	}
	return 0
}

func (m *LatestBlockData) GetToBlock() int64 {
	if m != nil {
		return m.ToBlock
	}
	return 0
}

func (m *LatestBlockData) GetSpecificBlock() int64 {
	if m != nil {
		return m.SpecificBlock
	}
	return 0
}

func (m *LatestBlockData) Reset()        { *m = LatestBlockData{} }
func (m *LatestBlockData) ProtoMessage() {}
func (m *LatestBlockData) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *LatestBlockData) Marshal() ([]byte, error) { return json.Marshal(m) }
func (m *LatestBlockData) Unmarshal(b []byte) error { return json.Unmarshal(b, m) }

// GetLatestBlockNumResponse is the response for GetLatestBlockNum.
type GetLatestBlockNumResponse struct {
	Block     uint64 `json:"block,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

func (m *GetLatestBlockNumResponse) GetBlock() uint64 {
	if m != nil {
		return m.Block
	}
	return 0
}

func (m *GetLatestBlockNumResponse) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *GetLatestBlockNumResponse) Reset()        { *m = GetLatestBlockNumResponse{} }
func (m *GetLatestBlockNumResponse) ProtoMessage() {}
func (m *GetLatestBlockNumResponse) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *GetLatestBlockNumResponse) Marshal() ([]byte, error) { return json.Marshal(m) }
func (m *GetLatestBlockNumResponse) Unmarshal(b []byte) error { return json.Unmarshal(b, m) }

// LatestBlockDataResponse is the response for GetLatestBlockData.
type LatestBlockDataResponse struct {
	LatestBlock     int64         `json:"latestBlock,omitempty"`
	RequestedHashes []*BlockStore `json:"requestedHashes,omitempty"`
}

func (m *LatestBlockDataResponse) GetLatestBlock() int64 {
	if m != nil {
		return m.LatestBlock
	}
	return 0
}

func (m *LatestBlockDataResponse) GetRequestedHashes() []*BlockStore {
	if m != nil {
		return m.RequestedHashes
	}
	return nil
}

func (m *LatestBlockDataResponse) Reset()        { *m = LatestBlockDataResponse{} }
func (m *LatestBlockDataResponse) ProtoMessage() {}
func (m *LatestBlockDataResponse) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *LatestBlockDataResponse) Marshal() ([]byte, error) { return json.Marshal(m) }
func (m *LatestBlockDataResponse) Unmarshal(b []byte) error { return json.Unmarshal(b, m) }

// BlockStore holds a block number and its hash.
type BlockStore struct {
	Block int64  `json:"block,omitempty"`
	Hash  string `json:"hash,omitempty"`
}

func (m *BlockStore) GetBlock() int64 {
	if m != nil {
		return m.Block
	}
	return 0
}

func (m *BlockStore) GetHash() string {
	if m != nil {
		return m.Hash
	}
	return ""
}

func (m *BlockStore) Reset()        { *m = BlockStore{} }
func (m *BlockStore) ProtoMessage() {}
func (m *BlockStore) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *BlockStore) Marshal() ([]byte, error) { return json.Marshal(m) }
func (m *BlockStore) Unmarshal(b []byte) error { return json.Unmarshal(b, m) }
