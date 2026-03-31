package relay

// BlockHashToHeight maps a block hash to its height, used in cache entries.
type BlockHashToHeight struct {
	Hash   string `json:"hash"`
	Height int64  `json:"height"`
}

func (b *BlockHashToHeight) GetHash() string {
	if b != nil {
		return b.Hash
	}
	return ""
}

func (b *BlockHashToHeight) GetHeight() int64 {
	if b != nil {
		return b.Height
	}
	return 0
}

// CacheRelayReply is the value stored in the relay cache for a given request.
type CacheRelayReply struct {
	Reply                 *RelayReply          `json:"reply"`
	OptionalMetadata      []Metadata           `json:"optional_metadata"`
	SeenBlock             int64                `json:"seen_block"`
	BlocksHashesToHeights []*BlockHashToHeight `json:"blocks_hashes_to_heights"`
}

func (c *CacheRelayReply) GetReply() *RelayReply {
	if c != nil {
		return c.Reply
	}
	return nil
}

func (c *CacheRelayReply) GetOptionalMetadata() []Metadata {
	if c != nil {
		return c.OptionalMetadata
	}
	return nil
}

func (c *CacheRelayReply) GetSeenBlock() int64 {
	if c != nil {
		return c.SeenBlock
	}
	return 0
}

func (c *CacheRelayReply) GetBlocksHashesToHeights() []*BlockHashToHeight {
	if c != nil {
		return c.BlocksHashesToHeights
	}
	return nil
}

// RelayCacheGet is the request message sent to the relay cache service.
type RelayCacheGet struct {
	RequestHash           []byte               `json:"request_hash"`
	BlockHash             []byte               `json:"block_hash"`
	Finalized             bool                 `json:"finalized"`
	RequestedBlock        int64                `json:"requested_block"`
	SharedStateId         string               `json:"shared_state_id"`
	ChainId               string               `json:"chain_id"`
	SeenBlock             int64                `json:"seen_block"`
	BlocksHashesToHeights []*BlockHashToHeight `json:"blocks_hashes_to_heights"`
}

func (r *RelayCacheGet) GetRequestHash() []byte {
	if r != nil {
		return r.RequestHash
	}
	return nil
}

func (r *RelayCacheGet) GetBlockHash() []byte {
	if r != nil {
		return r.BlockHash
	}
	return nil
}

func (r *RelayCacheGet) GetFinalized() bool {
	if r != nil {
		return r.Finalized
	}
	return false
}

func (r *RelayCacheGet) GetRequestedBlock() int64 {
	if r != nil {
		return r.RequestedBlock
	}
	return 0
}

func (r *RelayCacheGet) GetSharedStateId() string {
	if r != nil {
		return r.SharedStateId
	}
	return ""
}

func (r *RelayCacheGet) GetChainId() string {
	if r != nil {
		return r.ChainId
	}
	return ""
}

func (r *RelayCacheGet) GetSeenBlock() int64 {
	if r != nil {
		return r.SeenBlock
	}
	return 0
}

func (r *RelayCacheGet) GetBlocksHashesToHeights() []*BlockHashToHeight {
	if r != nil {
		return r.BlocksHashesToHeights
	}
	return nil
}

// RelayCacheSet is the request message sent to the relay cache service to store an entry.
type RelayCacheSet struct {
	RequestHash           []byte               `json:"request_hash"`
	BlockHash             []byte               `json:"block_hash"`
	Response              *RelayReply          `json:"response"`
	Finalized             bool                 `json:"finalized"`
	OptionalMetadata      []Metadata           `json:"optional_metadata"`
	SharedStateId         string               `json:"shared_state_id"`
	RequestedBlock        int64                `json:"requested_block"`
	ChainId               string               `json:"chain_id"`
	SeenBlock             int64                `json:"seen_block"`
	AverageBlockTime      int64                `json:"average_block_time"`
	IsNodeError           bool                 `json:"is_node_error"`
	BlocksHashesToHeights []*BlockHashToHeight `json:"blocks_hashes_to_heights"`
}

func (r *RelayCacheSet) GetRequestHash() []byte {
	if r != nil {
		return r.RequestHash
	}
	return nil
}

func (r *RelayCacheSet) GetBlockHash() []byte {
	if r != nil {
		return r.BlockHash
	}
	return nil
}

func (r *RelayCacheSet) GetResponse() *RelayReply {
	if r != nil {
		return r.Response
	}
	return nil
}

func (r *RelayCacheSet) GetFinalized() bool {
	if r != nil {
		return r.Finalized
	}
	return false
}

func (r *RelayCacheSet) GetOptionalMetadata() []Metadata {
	if r != nil {
		return r.OptionalMetadata
	}
	return nil
}

func (r *RelayCacheSet) GetSharedStateId() string {
	if r != nil {
		return r.SharedStateId
	}
	return ""
}

func (r *RelayCacheSet) GetRequestedBlock() int64 {
	if r != nil {
		return r.RequestedBlock
	}
	return 0
}

func (r *RelayCacheSet) GetChainId() string {
	if r != nil {
		return r.ChainId
	}
	return ""
}

func (r *RelayCacheSet) GetSeenBlock() int64 {
	if r != nil {
		return r.SeenBlock
	}
	return 0
}

func (r *RelayCacheSet) GetAverageBlockTime() int64 {
	if r != nil {
		return r.AverageBlockTime
	}
	return 0
}

func (r *RelayCacheSet) GetIsNodeError() bool {
	if r != nil {
		return r.IsNodeError
	}
	return false
}

func (r *RelayCacheSet) GetBlocksHashesToHeights() []*BlockHashToHeight {
	if r != nil {
		return r.BlocksHashesToHeights
	}
	return nil
}

// CacheUsage reports hit and miss statistics for the relay cache.
type CacheUsage struct {
	CacheHits   uint64 `json:"CacheHits"`
	CacheMisses uint64 `json:"CacheMisses"`
}

func (c *CacheUsage) GetCacheHits() uint64 {
	if c != nil {
		return c.CacheHits
	}
	return 0
}

func (c *CacheUsage) GetCacheMisses() uint64 {
	if c != nil {
		return c.CacheMisses
	}
	return 0
}

// CacheHash is a composite key used when computing the cache lookup hash.
type CacheHash struct {
	Request *RelayPrivateData `json:"request"`
	ChainId string            `json:"chain_id"`
}

func (c *CacheHash) GetRequest() *RelayPrivateData {
	if c != nil {
		return c.Request
	}
	return nil
}

func (c *CacheHash) GetChainId() string {
	if c != nil {
		return c.ChainId
	}
	return ""
}
