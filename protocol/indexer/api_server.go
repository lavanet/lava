package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/lavanet/lava/v5/utils"
)

// APIServer provides a REST API for querying indexed data
type APIServer struct {
	config  *IndexerConfig
	storage Storage
	metrics *IndexerMetrics
	server  *http.Server
}

// NewAPIServer creates a new API server
func NewAPIServer(config *IndexerConfig, storage Storage, metrics *IndexerMetrics) *APIServer {
	return &APIServer{
		config:  config,
		storage: storage,
		metrics: metrics,
	}
}

// Start starts the API server
func (api *APIServer) Start(ctx context.Context) error {
	if !api.config.EnableAPI {
		utils.LavaFormatInfo("API server disabled")
		return nil
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", api.handleHealth)

	// Metrics
	mux.HandleFunc("/metrics", api.handleMetrics)

	// Block endpoints
	mux.HandleFunc("/api/v1/blocks", api.handleGetBlocks)
	mux.HandleFunc("/api/v1/blocks/", api.handleGetBlock)
	mux.HandleFunc("/api/v1/blocks/latest", api.handleGetLatestBlock)

	// Transaction endpoints
	mux.HandleFunc("/api/v1/transactions", api.handleGetTransactions)
	mux.HandleFunc("/api/v1/transactions/", api.handleGetTransaction)

	// Log endpoints
	mux.HandleFunc("/api/v1/logs", api.handleGetLogs)

	// Decoded events endpoints
	mux.HandleFunc("/api/v1/events", api.handleGetDecodedEvents)
	mux.HandleFunc("/api/v1/events/types", api.handleGetEventTypes)

	// Internal transactions endpoints
	mux.HandleFunc("/api/v1/internal-transactions", api.handleGetInternalTransactions)

	// Contract watch endpoints
	mux.HandleFunc("/api/v1/watches", api.handleGetWatches)

	// Status endpoint
	mux.HandleFunc("/api/v1/status", api.handleGetStatus)

	api.server = &http.Server{
		Addr:    api.config.APIListenAddr,
		Handler: mux,
	}

	utils.LavaFormatInfo("API server starting", utils.LogAttr("address", api.config.APIListenAddr))

	go func() {
		if err := api.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.LavaFormatError("API server error", err)
		}
	}()

	return nil
}

// Stop stops the API server
func (api *APIServer) Stop(ctx context.Context) error {
	if api.server == nil {
		return nil
	}

	utils.LavaFormatInfo("Stopping API server")
	return api.server.Shutdown(ctx)
}

// HTTP Handlers

func (api *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (api *APIServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api.metrics.GetStats())
}

func (api *APIServer) handleGetBlocks(w http.ResponseWriter, r *http.Request) {
	query := &BlockQuery{
		ChainID:     api.config.ChainID,
		Limit:       100, // Default
		OrderByDesc: true,
	}

	// Parse query parameters
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if from, err := strconv.ParseInt(fromStr, 10, 64); err == nil {
			query.FromBlock = &from
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if to, err := strconv.ParseInt(toStr, 10, 64); err == nil {
			query.ToBlock = &to
		}
	}
	if miner := r.URL.Query().Get("miner"); miner != "" {
		query.Miner = &miner
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			query.Limit = limit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			query.Offset = offset
		}
	}

	blocks, err := api.storage.GetBlocks(r.Context(), query)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to query blocks", err)
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"blocks": blocks,
		"count":  len(blocks),
	})
}

func (api *APIServer) handleGetBlock(w http.ResponseWriter, r *http.Request) {
	// Extract block number from path
	blockNumStr := r.URL.Path[len("/api/v1/blocks/"):]
	blockNum, err := strconv.ParseInt(blockNumStr, 10, 64)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid block number", err)
		return
	}

	block, err := api.storage.GetBlock(r.Context(), api.config.ChainID, blockNum)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to get block", err)
		return
	}

	if block == nil {
		api.writeError(w, http.StatusNotFound, "Block not found", nil)
		return
	}

	api.writeJSON(w, http.StatusOK, block)
}

func (api *APIServer) handleGetLatestBlock(w http.ResponseWriter, r *http.Request) {
	block, err := api.storage.GetLatestBlock(r.Context(), api.config.ChainID)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to get latest block", err)
		return
	}

	if block == nil {
		api.writeError(w, http.StatusNotFound, "No blocks indexed", nil)
		return
	}

	api.writeJSON(w, http.StatusOK, block)
}

func (api *APIServer) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	query := &TransactionQuery{
		ChainID:     api.config.ChainID,
		Limit:       100,
		OrderByDesc: true,
	}

	if fromStr := r.URL.Query().Get("from_block"); fromStr != "" {
		if from, err := strconv.ParseInt(fromStr, 10, 64); err == nil {
			query.FromBlock = &from
		}
	}
	if toStr := r.URL.Query().Get("to_block"); toStr != "" {
		if to, err := strconv.ParseInt(toStr, 10, 64); err == nil {
			query.ToBlock = &to
		}
	}
	if fromAddr := r.URL.Query().Get("from"); fromAddr != "" {
		query.FromAddress = &fromAddr
	}
	if toAddr := r.URL.Query().Get("to"); toAddr != "" {
		query.ToAddress = &toAddr
	}
	if contract := r.URL.Query().Get("contract"); contract != "" {
		query.ContractAddr = &contract
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			query.Limit = limit
		}
	}

	txs, err := api.storage.GetTransactions(r.Context(), query)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to query transactions", err)
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": txs,
		"count":        len(txs),
	})
}

func (api *APIServer) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	txHash := r.URL.Path[len("/api/v1/transactions/"):]
	if txHash == "" {
		api.writeError(w, http.StatusBadRequest, "Transaction hash required", nil)
		return
	}

	tx, err := api.storage.GetTransaction(r.Context(), api.config.ChainID, txHash)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to get transaction", err)
		return
	}

	if tx == nil {
		api.writeError(w, http.StatusNotFound, "Transaction not found", nil)
		return
	}

	api.writeJSON(w, http.StatusOK, tx)
}

func (api *APIServer) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	query := &LogQuery{
		ChainID:     api.config.ChainID,
		Limit:       100,
		OrderByDesc: true,
	}

	if contract := r.URL.Query().Get("contract"); contract != "" {
		query.ContractAddress = &contract
	}
	if fromStr := r.URL.Query().Get("from_block"); fromStr != "" {
		if from, err := strconv.ParseInt(fromStr, 10, 64); err == nil {
			query.FromBlock = &from
		}
	}
	if toStr := r.URL.Query().Get("to_block"); toStr != "" {
		if to, err := strconv.ParseInt(toStr, 10, 64); err == nil {
			query.ToBlock = &to
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			query.Limit = limit
		}
	}

	// Parse topic filters
	topics := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		if topic := r.URL.Query().Get(fmt.Sprintf("topic%d", i)); topic != "" {
			topics = append(topics, topic)
		} else {
			topics = append(topics, "")
		}
	}
	query.Topics = topics

	logs, err := api.storage.GetLogs(r.Context(), query)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to query logs", err)
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

func (api *APIServer) handleGetWatches(w http.ResponseWriter, r *http.Request) {
	watches, err := api.storage.GetContractWatches(r.Context(), api.config.ChainID, true)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to get watches", err)
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"watches": watches,
		"count":   len(watches),
	})
}

func (api *APIServer) handleGetDecodedEvents(w http.ResponseWriter, r *http.Request) {
	var contractAddr *string
	if addr := r.URL.Query().Get("contract"); addr != "" {
		contractAddr = &addr
	}

	var eventType *string
	if et := r.URL.Query().Get("event_type"); et != "" {
		eventType = &et
	}

	var fromBlock, toBlock *int64
	if fromStr := r.URL.Query().Get("from_block"); fromStr != "" {
		if from, err := strconv.ParseInt(fromStr, 10, 64); err == nil {
			fromBlock = &from
		}
	}
	if toStr := r.URL.Query().Get("to_block"); toStr != "" {
		if to, err := strconv.ParseInt(toStr, 10, 64); err == nil {
			toBlock = &to
		}
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	sqlStorage, ok := api.storage.(*SQLiteStorage)
	if !ok {
		api.writeError(w, http.StatusInternalServerError, "Storage type not supported", nil)
		return
	}

	events, err := sqlStorage.GetDecodedEvents(r.Context(), api.config.ChainID, contractAddr, eventType, fromBlock, toBlock, limit)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to query decoded events", err)
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (api *APIServer) handleGetEventTypes(w http.ResponseWriter, r *http.Request) {
	eventTypes := map[string]string{
		string(EventTypeTransfer):       GetEventTypeInfo(EventTypeTransfer),
		string(EventTypeApproval):       GetEventTypeInfo(EventTypeApproval),
		string(EventTypeApprovalForAll): GetEventTypeInfo(EventTypeApprovalForAll),
		string(EventTypeSwap):           GetEventTypeInfo(EventTypeSwap),
		string(EventTypeMint):           GetEventTypeInfo(EventTypeMint),
		string(EventTypeBurn):           GetEventTypeInfo(EventTypeBurn),
		string(EventTypeCustom):         GetEventTypeInfo(EventTypeCustom),
		string(EventTypeUnknown):        GetEventTypeInfo(EventTypeUnknown),
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"event_types": eventTypes,
		"signatures": map[string]string{
			"Transfer":       EventSignatureTransfer,
			"Approval":       EventSignatureApproval,
			"ApprovalForAll": EventSignatureApprovalForAll,
			"Swap":           EventSignatureSwap,
			"Mint":           EventSignatureMint,
			"Burn":           EventSignatureBurn,
		},
	})
}

func (api *APIServer) handleGetInternalTransactions(w http.ResponseWriter, r *http.Request) {
	txHash := r.URL.Query().Get("tx_hash")
	if txHash == "" {
		api.writeError(w, http.StatusBadRequest, "tx_hash parameter required", nil)
		return
	}

	sqlStorage, ok := api.storage.(*SQLiteStorage)
	if !ok {
		api.writeError(w, http.StatusInternalServerError, "Storage type not supported", nil)
		return
	}

	internalTxs, err := sqlStorage.GetInternalTransactions(r.Context(), api.config.ChainID, txHash)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to get internal transactions", err)
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"internal_transactions": internalTxs,
		"count":                 len(internalTxs),
	})
}

func (api *APIServer) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	state, err := api.storage.GetIndexerState(r.Context(), api.config.ChainID)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to get state", err)
		return
	}

	status := map[string]interface{}{
		"chain_id": api.config.ChainID,
		"metrics":  api.metrics.GetStats(),
		"features": map[string]bool{
			"index_transactions":  api.config.IndexTransactions,
			"index_internal_txs":  api.config.IndexInternalTxs,
			"index_logs":          api.config.IndexLogs,
			"decode_events":       api.config.DecodeEvents,
			"track_common_events": api.config.TrackCommonEvents,
		},
	}

	if state != nil {
		status["last_indexed_block"] = state.LastBlockNumber
		status["last_indexed_hash"] = state.LastBlockHash
		status["last_updated"] = state.UpdatedAt
	}

	api.writeJSON(w, http.StatusOK, status)
}

// Helper methods

func (api *APIServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		utils.LavaFormatError("Failed to write JSON response", err)
	}
}

func (api *APIServer) writeError(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		utils.LavaFormatWarning(message, err)
	}

	api.writeJSON(w, status, map[string]string{
		"error": message,
		"details": func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
	})
}
