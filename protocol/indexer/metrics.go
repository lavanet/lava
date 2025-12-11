package indexer

import (
	"sync/atomic"
	"time"
)

// IndexerMetrics tracks indexing metrics
type IndexerMetrics struct {
	blocksIndexed     atomic.Int64
	logsIndexed       atomic.Int64
	txsIndexed        atomic.Int64
	indexErrors       atomic.Int64
	lastIndexedBlock  atomic.Int64
	lastIndexDuration atomic.Int64 // nanoseconds
}

// NewIndexerMetrics creates a new metrics instance
func NewIndexerMetrics() *IndexerMetrics {
	return &IndexerMetrics{}
}

// IncBlocksIndexed increments the blocks indexed counter
func (m *IndexerMetrics) IncBlocksIndexed() {
	m.blocksIndexed.Add(1)
}

// IncLogsIndexed increments the logs indexed counter
func (m *IndexerMetrics) IncLogsIndexed() {
	m.logsIndexed.Add(1)
}

// IncTransactionsIndexed increments the transactions indexed counter
func (m *IndexerMetrics) IncTransactionsIndexed() {
	m.txsIndexed.Add(1)
}

// IncIndexErrors increments the index errors counter
func (m *IndexerMetrics) IncIndexErrors() {
	m.indexErrors.Add(1)
}

// SetLastIndexedBlock sets the last indexed block number
func (m *IndexerMetrics) SetLastIndexedBlock(blockNum int64) {
	m.lastIndexedBlock.Store(blockNum)
}

// ObserveBlockIndexDuration records the duration of indexing a block
func (m *IndexerMetrics) ObserveBlockIndexDuration(duration time.Duration) {
	m.lastIndexDuration.Store(int64(duration))
}

// GetBlocksIndexed returns the total blocks indexed
func (m *IndexerMetrics) GetBlocksIndexed() int64 {
	return m.blocksIndexed.Load()
}

// GetLogsIndexed returns the total logs indexed
func (m *IndexerMetrics) GetLogsIndexed() int64 {
	return m.logsIndexed.Load()
}

// GetTransactionsIndexed returns the total transactions indexed
func (m *IndexerMetrics) GetTransactionsIndexed() int64 {
	return m.txsIndexed.Load()
}

// GetIndexErrors returns the total index errors
func (m *IndexerMetrics) GetIndexErrors() int64 {
	return m.indexErrors.Load()
}

// GetLastIndexedBlock returns the last indexed block number
func (m *IndexerMetrics) GetLastIndexedBlock() int64 {
	return m.lastIndexedBlock.Load()
}

// GetLastIndexDuration returns the last index duration
func (m *IndexerMetrics) GetLastIndexDuration() time.Duration {
	return time.Duration(m.lastIndexDuration.Load())
}

// GetStats returns all metrics as a map
func (m *IndexerMetrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"blocks_indexed":         m.GetBlocksIndexed(),
		"logs_indexed":           m.GetLogsIndexed(),
		"transactions_indexed":   m.GetTransactionsIndexed(),
		"index_errors":           m.GetIndexErrors(),
		"last_indexed_block":     m.GetLastIndexedBlock(),
		"last_index_duration_ms": m.GetLastIndexDuration().Milliseconds(),
	}
}
