package rpcsmartrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	gojson "github.com/goccy/go-json"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	rpcclient "github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

// Note: Configuration constants are now in WebsocketConfig (websocket_config.go)
// Legacy constants kept for backwards compatibility
const (
	// DefaultMaxSubscriptionsPerClient is the default max subscriptions per client
	DefaultMaxSubscriptionsPerClient = 25

	// DefaultCleanupInterval is the default interval for periodic cleanup
	DefaultCleanupInterval = 1 * time.Minute

	// DefaultMaxMessageSize is the default maximum message size (1 MB)
	DefaultMaxMessageSize = 1048576
)

// directActiveSubscription holds state for an active upstream subscription
type directActiveSubscription struct {
	// Upstream connection info
	upstreamPool         *UpstreamWSPool
	upstreamConnection   *UpstreamWSConnection // The specific connection used for this subscription
	upstreamSubscription *rpcclient.ClientSubscription
	upstreamID           string // ID returned by upstream node

	// Router-generated info - each client gets their own router ID
	// that maps to the shared upstream ID. This enables proper deduplication
	// where multiple clients can share one upstream subscription while each
	// having their own unique subscription ID for unsubscribe operations.
	clientRouterIDs map[string]string // clientKey -> routerID
	hashedParams    string            // Hash of subscription parameters

	// Subscription params for restoration after reconnect
	subscriptionParams []byte  // Original subscription params (JSON)
	subscribeMethod    string  // Method name (e.g., "eth_subscribe", "subscribe")

	// Client tracking - multiple clients can share one upstream subscription
	connectedClients map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]

	// First reply cached for deduplication
	firstReply *pairingtypes.RelayReply

	// Original result from upstream for Tendermint (stores {"query":"..."} format)
	// Used when joining clients need the original response format
	originalResult json.RawMessage

	// Lifecycle
	ctx          context.Context
	cancel       context.CancelFunc
	closeSubChan chan struct{}
	messagesChan chan *rpcclient.JsonrpcMessage

	// Restoration state
	restoring atomic.Bool // Prevents concurrent restoration attempts
}

// pendingSubscriptionsBroadcastManager handles synchronization for concurrent subscription requests
type pendingSubscriptionsBroadcastManager struct {
	broadcastChannelList []chan bool
}

func (psbm *pendingSubscriptionsBroadcastManager) broadcastToChannelList(value bool) {
	for _, ch := range psbm.broadcastChannelList {
		ch <- value
	}
}

// DirectWSSubscriptionManager manages WebSocket subscriptions directly to upstream endpoints
// without going through Lava providers. It implements chainlib.WSSubscriptionManager.
//
// This follows the same patterns as ConsumerWSSubscriptionManager but connects directly
// to RPC endpoints instead of routing through providers.
type DirectWSSubscriptionManager struct {
	// Client-facing (reuse existing patterns)
	connectedClients map[string]map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]
	// First key: clientKey (dappID:ip:wsUID), Second key: hashedParams

	// Active subscriptions keyed by hashedParams
	activeSubscriptions map[string]*directActiveSubscription

	// Pending subscriptions for deduplication
	pendingSubscriptions map[string]*pendingSubscriptionsBroadcastManager

	// Upstream connection pools keyed by endpoint URL
	upstreamPools map[string]*UpstreamWSPool

	// Subscription ID mapping
	idMapper *SubscriptionIDMapper

	// Dependencies
	metricsManager *metrics.ConsumerMetricsManager
	connectionType string
	chainID        string
	apiInterface   string

	// Upstream endpoint configuration - multiple endpoints for optimizer selection
	wsEndpoints    []*common.NodeUrl            // All available WebSocket endpoints
	endpointsByURL map[string]*common.NodeUrl   // Quick lookup by URL
	optimizer      WebSocketEndpointOptimizer   // Optimizer for endpoint selection (can be nil)

	// Sticky sessions for subscription affinity - same client uses same endpoint
	stickyStore *lavasession.StickySessionStore

	// Configuration - all configurable parameters
	config *WebsocketConfig

	// Rate limiting - per-client subscription rate limits
	rateLimiter *ClientRateLimiter

	// Total subscription counter for global limit tracking
	totalSubscriptions atomic.Int64

	lock sync.RWMutex
}

// WebSocketEndpointOptimizer is an interface for selecting WebSocket endpoints
// This allows using ProviderOptimizer or a simple round-robin fallback
type WebSocketEndpointOptimizer interface {
	// ChooseProvider selects the best endpoint(s) from available addresses
	ChooseProvider(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string, tier int)
	// AppendRelayData updates metrics after successful relay
	AppendRelayData(provider string, latency time.Duration, cu, syncBlock uint64)
	// AppendRelayFailure updates metrics after failed relay
	AppendRelayFailure(providerAddress string)
}

// NewDirectWSSubscriptionManager creates a new direct WebSocket subscription manager.
// If config is nil, DefaultWebsocketConfig() will be used.
func NewDirectWSSubscriptionManager(
	metricsManager *metrics.ConsumerMetricsManager,
	connectionType string,
	chainID string,
	apiInterface string,
	wsEndpoints []*common.NodeUrl,
	optimizer WebSocketEndpointOptimizer,
	config *WebsocketConfig,
) *DirectWSSubscriptionManager {
	// Build URL lookup map
	endpointsByURL := make(map[string]*common.NodeUrl, len(wsEndpoints))
	for _, ep := range wsEndpoints {
		endpointsByURL[ep.Url] = ep
	}

	// Use default config if none provided
	if config == nil {
		config = DefaultWebsocketConfig()
	}

	return &DirectWSSubscriptionManager{
		connectedClients:     make(map[string]map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		activeSubscriptions:  make(map[string]*directActiveSubscription),
		pendingSubscriptions: make(map[string]*pendingSubscriptionsBroadcastManager),
		upstreamPools:        make(map[string]*UpstreamWSPool),
		idMapper:             NewSubscriptionIDMapper(),
		metricsManager:       metricsManager,
		connectionType:       connectionType,
		chainID:              chainID,
		apiInterface:         apiInterface,
		wsEndpoints:          wsEndpoints,
		endpointsByURL:       endpointsByURL,
		optimizer:            optimizer,
		stickyStore:          lavasession.NewStickySessionStore(),
		config:               config,
		rateLimiter:          NewClientRateLimiter(config),
	}
}

// Start starts the background cleanup goroutine for the DirectWSSubscriptionManager.
// This should be called once after creating the manager.
func (dwsm *DirectWSSubscriptionManager) Start(ctx context.Context) {
	go dwsm.cleanupStaleSubscriptions(ctx)
	utils.LavaFormatInfo("DirectWS: started subscription manager",
		utils.LogAttr("chainID", dwsm.chainID),
		utils.LogAttr("cleanupInterval", dwsm.config.CleanupInterval),
		utils.LogAttr("maxSubsPerClient", dwsm.config.MaxSubscriptionsPerClient),
		utils.LogAttr("maxTotalSubscriptions", dwsm.config.MaxTotalSubscriptions),
		utils.LogAttr("subscriptionsPerMinutePerClient", dwsm.config.SubscriptionsPerMinutePerClient),
	)
}

// cleanupStaleSubscriptions periodically removes stale subscriptions with cancelled contexts
// This matches rpcconsumer's cleanup pattern (interval from config, default 1 minute)
func (dwsm *DirectWSSubscriptionManager) cleanupStaleSubscriptions(ctx context.Context) {
	ticker := time.NewTicker(dwsm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			utils.LavaFormatDebug("DirectWS: cleanup goroutine stopped")
			return
		case <-ticker.C:
			dwsm.performCleanup()
		}
	}
}

// performCleanup removes subscriptions with cancelled contexts and cleans up orphaned clients
func (dwsm *DirectWSSubscriptionManager) performCleanup() {
	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	removedSubs := 0
	removedClients := 0

	// Clean up subscriptions with no connected clients or cancelled contexts
	for hashedParams, activeSub := range dwsm.activeSubscriptions {
		// Check if subscription context is done
		select {
		case <-activeSub.ctx.Done():
			// Context cancelled, clean up
			utils.LavaFormatDebug("DirectWS: cleaning up subscription with cancelled context",
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
			activeSub.upstreamSubscription.Unsubscribe()
			dwsm.idMapper.RemoveAllForUpstream(activeSub.upstreamID)
			delete(dwsm.activeSubscriptions, hashedParams)
			removedSubs++
			continue
		default:
			// Context still active
		}

		// Remove clients with closed/cancelled contexts
		// Note: We can't check if SafeChannelSender is closed directly,
		// so we rely on the subscription context and client disconnect handlers

		// If no clients left, close the subscription
		if len(activeSub.connectedClients) == 0 {
			utils.LavaFormatDebug("DirectWS: cleaning up subscription with no clients",
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
			activeSub.upstreamSubscription.Unsubscribe()
			activeSub.cancel()
			dwsm.idMapper.RemoveAllForUpstream(activeSub.upstreamID)
			delete(dwsm.activeSubscriptions, hashedParams)
			removedSubs++
		}
	}

	// Clean up orphaned client entries (clients with no subscriptions)
	for clientKey, subs := range dwsm.connectedClients {
		// Check if subscriptions still exist in activeSubscriptions
		for hashedParams := range subs {
			if _, exists := dwsm.activeSubscriptions[hashedParams]; !exists {
				delete(subs, hashedParams)
			}
		}
		if len(subs) == 0 {
			delete(dwsm.connectedClients, clientKey)
			dwsm.stickyStore.Delete(clientKey)
			removedClients++
		}
	}

	// Log warning if subscription count is high (match rpcconsumer pattern)
	totalSubs := len(dwsm.activeSubscriptions)
	if totalSubs > dwsm.config.MaxTotalSubscriptions {
		utils.LavaFormatWarning("DirectWS: subscription count high, potential memory issue", nil,
			utils.LogAttr("current", totalSubs),
			utils.LogAttr("threshold", dwsm.config.MaxTotalSubscriptions),
		)
	}

	if removedSubs > 0 || removedClients > 0 {
		utils.LavaFormatDebug("DirectWS: cleanup completed",
			utils.LogAttr("removedSubscriptions", removedSubs),
			utils.LogAttr("removedClients", removedClients),
			utils.LogAttr("activeSubscriptions", len(dwsm.activeSubscriptions)),
			utils.LogAttr("connectedClients", len(dwsm.connectedClients)),
		)
	}
}

// selectEndpoint selects the best WebSocket endpoint for a client.
// Selection priority:
//  1. Sticky session (if client already has affinity to an endpoint)
//  2. Optimizer-based selection (QoS, latency, etc.)
//  3. First available endpoint (fallback)
func (dwsm *DirectWSSubscriptionManager) selectEndpoint(clientKey string, ignoredEndpoints map[string]struct{}) (*common.NodeUrl, error) {
	if len(dwsm.wsEndpoints) == 0 {
		return nil, fmt.Errorf("no WebSocket endpoints configured")
	}

	// Priority 1: Check sticky session for this client
	if clientKey != "" {
		if stickySession, exists := dwsm.stickyStore.Get(clientKey); exists {
			stickyEndpoint, found := dwsm.endpointsByURL[stickySession.Provider]
			if found {
				// Check if sticky endpoint is not ignored
				if ignoredEndpoints == nil {
					utils.LavaFormatDebug("DirectWS: using sticky session endpoint",
						utils.LogAttr("clientKey", clientKey),
						utils.LogAttr("endpoint", sanitizeEndpointURL(stickySession.Provider)),
					)
					return stickyEndpoint, nil
				}
				if _, ignored := ignoredEndpoints[stickySession.Provider]; !ignored {
					utils.LavaFormatDebug("DirectWS: using sticky session endpoint",
						utils.LogAttr("clientKey", clientKey),
						utils.LogAttr("endpoint", sanitizeEndpointURL(stickySession.Provider)),
					)
					return stickyEndpoint, nil
				}
				// Sticky endpoint is ignored, clear it and continue to optimizer
				utils.LavaFormatDebug("DirectWS: sticky endpoint ignored, clearing affinity",
					utils.LogAttr("clientKey", clientKey),
					utils.LogAttr("ignoredEndpoint", sanitizeEndpointURL(stickySession.Provider)),
				)
				dwsm.stickyStore.Delete(clientKey)
			}
		}
	}

	// Priority 2: If only one endpoint or no optimizer, use first available
	if len(dwsm.wsEndpoints) == 1 || dwsm.optimizer == nil {
		for _, ep := range dwsm.wsEndpoints {
			if ignoredEndpoints == nil {
				return ep, nil
			}
			if _, ignored := ignoredEndpoints[ep.Url]; !ignored {
				return ep, nil
			}
		}
		return nil, fmt.Errorf("all WebSocket endpoints are ignored/unavailable")
	}

	// Priority 3: Use optimizer to select best endpoint
	allURLs := make([]string, 0, len(dwsm.wsEndpoints))
	for _, ep := range dwsm.wsEndpoints {
		allURLs = append(allURLs, ep.Url)
	}

	// cu=1 and requestedBlock=LATEST_BLOCK are sensible defaults for subscriptions
	selectedURLs, tier := dwsm.optimizer.ChooseProvider(allURLs, ignoredEndpoints, 1, -2) // -2 = LATEST_BLOCK

	if len(selectedURLs) == 0 {
		// Optimizer returned nothing, fall back to first non-ignored
		for _, ep := range dwsm.wsEndpoints {
			if ignoredEndpoints == nil {
				return ep, nil
			}
			if _, ignored := ignoredEndpoints[ep.Url]; !ignored {
				return ep, nil
			}
		}
		return nil, fmt.Errorf("optimizer returned no endpoints and all fallbacks are ignored")
	}

	selectedURL := selectedURLs[0]
	selectedEndpoint, exists := dwsm.endpointsByURL[selectedURL]
	if !exists {
		return nil, fmt.Errorf("optimizer selected unknown endpoint: %s", selectedURL)
	}

	utils.LavaFormatDebug("DirectWS: selected endpoint via optimizer",
		utils.LogAttr("endpoint", sanitizeEndpointURL(selectedURL)),
		utils.LogAttr("tier", tier),
		utils.LogAttr("totalEndpoints", len(dwsm.wsEndpoints)),
	)

	return selectedEndpoint, nil
}

// CreateWebSocketConnectionUniqueKey creates a unique key for a WebSocket connection
func (dwsm *DirectWSSubscriptionManager) CreateWebSocketConnectionUniqueKey(dappID, consumerIp, wsUID string) string {
	return dappID + ":" + consumerIp + ":" + wsUID
}

// GetOrCreatePool gets or creates an upstream WebSocket pool for the given endpoint.
// The pool is configured with WebSocket settings from the manager's config.
func (dwsm *DirectWSSubscriptionManager) GetOrCreatePool(nodeUrl *common.NodeUrl) *UpstreamWSPool {
	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	pool, exists := dwsm.upstreamPools[nodeUrl.Url]
	if !exists {
		pool = NewUpstreamWSPoolWithConfig(nodeUrl, dwsm.config)
		dwsm.upstreamPools[nodeUrl.Url] = pool
	}
	return pool
}

// StartSubscription implements chainlib.WSSubscriptionManager.
// It starts a new WebSocket subscription or joins an existing one.
func (dwsm *DirectWSSubscriptionManager) StartSubscription(
	ctx context.Context,
	protocolMessage chainlib.ProtocolMessage,
	dappID string,
	consumerIp string,
	webSocketConnectionUniqueId string,
	metricsData *metrics.RelayMetrics,
) (firstReply *pairingtypes.RelayReply, repliesChan <-chan *pairingtypes.RelayReply, err error) {
	// Extract hashed params from protocol message (same as ConsumerWSSubscriptionManager)
	hashedParams, subscriptionParams, err := dwsm.getHashedParams(protocolMessage)
	if err != nil {
		return nil, nil, utils.LavaFormatError("could not marshal params", err)
	}

	// Extract the subscription method from the API definition
	// - EVM chains use "eth_subscribe"
	// - Tendermint uses "subscribe"
	subscribeMethod := protocolMessage.GetApi().Name

	// Extract the request ID from the client's message - required for JSON-RPC 2.0 compliance
	// The response ID must match the request ID exactly
	requestID := extractRequestID(protocolMessage.RelayPrivateData().Data)

	clientKey := dwsm.CreateWebSocketConnectionUniqueKey(dappID, consumerIp, webSocketConnectionUniqueId)

	utils.LavaFormatTrace("DirectWS: request to start subscription",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("clientKey", clientKey),
		utils.LogAttr("subscribeMethod", subscribeMethod),
		utils.LogAttr("apiInterface", dwsm.apiInterface),
	)

	// Check rate limiting (subscriptions per minute per client)
	if !dwsm.rateLimiter.AllowSubscribe(clientKey) {
		utils.LavaFormatWarning("DirectWS: client rate limit exceeded", nil,
			utils.LogAttr("clientKey", clientKey),
			utils.LogAttr("limit", dwsm.config.SubscriptionsPerMinutePerClient),
		)
		go dwsm.metricsManager.SetFailedWsSubscriptionRequestMetric(dwsm.chainID, dwsm.apiInterface)
		return nil, nil, fmt.Errorf("subscription rate limit exceeded: max %d subscriptions per minute",
			dwsm.config.SubscriptionsPerMinutePerClient)
	}

	// Check per-client subscription limit
	dwsm.lock.RLock()
	clientSubs := dwsm.connectedClients[clientKey]
	currentSubCount := len(clientSubs)
	dwsm.lock.RUnlock()

	if currentSubCount >= dwsm.config.MaxSubscriptionsPerClient {
		if dwsm.config.ShouldRejectOnClientLimit() {
			utils.LavaFormatWarning("DirectWS: client subscription limit exceeded (rejecting)", nil,
				utils.LogAttr("clientKey", clientKey),
				utils.LogAttr("currentCount", currentSubCount),
				utils.LogAttr("limit", dwsm.config.MaxSubscriptionsPerClient),
			)
			go dwsm.metricsManager.SetFailedWsSubscriptionRequestMetric(dwsm.chainID, dwsm.apiInterface)
			return nil, nil, fmt.Errorf("subscription limit exceeded: client has %d subscriptions (max %d)",
				currentSubCount, dwsm.config.MaxSubscriptionsPerClient)
		}
		// Warn but continue (default behavior)
		utils.LavaFormatWarning("DirectWS: client subscription limit exceeded (warn only)", nil,
			utils.LogAttr("clientKey", clientKey),
			utils.LogAttr("currentCount", currentSubCount),
			utils.LogAttr("limit", dwsm.config.MaxSubscriptionsPerClient),
		)
	}

	// Check global subscription limit
	totalSubs := dwsm.totalSubscriptions.Load()
	if int(totalSubs) >= dwsm.config.MaxTotalSubscriptions {
		if dwsm.config.ShouldRejectOnTotalLimit() {
			utils.LavaFormatWarning("DirectWS: global subscription limit exceeded (rejecting)", nil,
				utils.LogAttr("currentTotal", totalSubs),
				utils.LogAttr("limit", dwsm.config.MaxTotalSubscriptions),
			)
			go dwsm.metricsManager.SetFailedWsSubscriptionRequestMetric(dwsm.chainID, dwsm.apiInterface)
			return nil, nil, fmt.Errorf("global subscription limit exceeded: %d subscriptions (max %d)",
				totalSubs, dwsm.config.MaxTotalSubscriptions)
		}
		// Warn but continue (default behavior)
		utils.LavaFormatWarning("DirectWS: global subscription limit exceeded (warn only)", nil,
			utils.LogAttr("currentTotal", totalSubs),
			utils.LogAttr("limit", dwsm.config.MaxTotalSubscriptions),
		)
	}

	// Create reply channel for this client
	repliesChannel := make(chan *pairingtypes.RelayReply)
	safeChannelSender := common.NewSafeChannelSender(ctx, repliesChannel)

	// Track metrics
	go dwsm.metricsManager.SetWsSubscriptionRequestMetric(dwsm.chainID, dwsm.apiInterface)

	// Check for existing active subscription
	existingReply, joined := dwsm.checkForActiveSubscriptionAndConnect(ctx, hashedParams, clientKey, safeChannelSender, requestID)
	if existingReply != nil {
		if joined {
			go dwsm.metricsManager.SetDuplicatedWsSubscriptionRequestMetric(dwsm.chainID, dwsm.apiInterface)
			return existingReply, repliesChannel, nil
		}
		// Already had this exact subscription
		safeChannelSender.Close()
		return existingReply, nil, nil
	}

	// Handle pending subscriptions (another client is creating the same subscription)
	for {
		pendingChan, foundPending := dwsm.checkAndAddPendingSubscription(hashedParams)
		if foundPending {
			utils.LavaFormatTrace("DirectWS: waiting for pending subscription")
			success := <-pendingChan
			if success {
				existingReply, joined := dwsm.checkForActiveSubscriptionAndConnect(ctx, hashedParams, clientKey, safeChannelSender, requestID)
				if existingReply != nil {
					if joined {
						return existingReply, repliesChannel, nil
					}
					safeChannelSender.Close()
					return existingReply, nil, nil
				}
			}
			// Failed, retry
			utils.LavaFormatDebug("DirectWS: pending subscription failed, retrying")
		} else {
			break
		}
	}

	// No active subscription found, create new one
	// Select best endpoint using sticky session, optimizer, or first available
	startTime := time.Now()
	selectedEndpoint, err := dwsm.selectEndpoint(clientKey, nil)
	if err != nil {
		dwsm.failPendingSubscription(hashedParams)
		go dwsm.metricsManager.SetFailedWsSubscriptionRequestMetric(dwsm.chainID, dwsm.apiInterface)
		return nil, nil, fmt.Errorf("failed to select WebSocket endpoint: %w", err)
	}

	utils.LavaFormatTrace("DirectWS: creating new upstream subscription",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("endpoint", sanitizeEndpointURL(selectedEndpoint.Url)),
	)

	// Get or create upstream pool for selected endpoint
	pool := dwsm.GetOrCreatePool(selectedEndpoint)

	// Get connection
	conn, err := pool.GetConnection(ctx)
	if err != nil {
		// Report failure to optimizer
		if dwsm.optimizer != nil {
			dwsm.optimizer.AppendRelayFailure(selectedEndpoint.Url)
		}
		dwsm.failPendingSubscription(hashedParams)
		go dwsm.metricsManager.SetFailedWsSubscriptionRequestMetric(dwsm.chainID, dwsm.apiInterface)
		return nil, nil, fmt.Errorf("failed to get WebSocket connection: %w", err)
	}

	// Create upstream subscription
	upstreamSub, firstMsg, msgChan, err := dwsm.createUpstreamSubscription(ctx, conn, subscriptionParams, subscribeMethod)
	if err != nil {
		// Report failure to optimizer
		if dwsm.optimizer != nil {
			dwsm.optimizer.AppendRelayFailure(selectedEndpoint.Url)
		}
		dwsm.failPendingSubscription(hashedParams)
		go dwsm.metricsManager.SetFailedWsSubscriptionRequestMetric(dwsm.chainID, dwsm.apiInterface)
		return nil, nil, fmt.Errorf("failed to create upstream subscription: %w", err)
	}

	// Report success to optimizer
	if dwsm.optimizer != nil {
		latency := time.Since(startTime)
		dwsm.optimizer.AppendRelayData(selectedEndpoint.Url, latency, 1, 0) // cu=1, syncBlock=0 (unknown)
	}

	// Store sticky session for client affinity
	// Future subscriptions from this client will use the same endpoint
	dwsm.stickyStore.Set(clientKey, &lavasession.StickySession{
		Provider: selectedEndpoint.Url,
		Epoch:    0, // Epoch is not used for direct RPC (no provider rotation)
	})
	utils.LavaFormatTrace("DirectWS: stored sticky session",
		utils.LogAttr("clientKey", clientKey),
		utils.LogAttr("endpoint", sanitizeEndpointURL(selectedEndpoint.Url)),
	)

	// Generate router subscription ID (for EVM deduplication)
	// For Tendermint, the query string is the identifier, but we still generate
	// router IDs for internal tracking (not exposed to clients)
	routerID := dwsm.idMapper.GenerateRouterID(clientKey)

	// Extract upstream subscription ID based on API interface
	// - EVM chains: subscription ID is in the response result (hex string)
	// - Tendermint: subscription ID is the query parameter from the request
	upstreamID := getSubscriptionID(dwsm.apiInterface, firstMsg, subscriptionParams)
	dwsm.idMapper.RegisterMapping(routerID, upstreamID)

	// Store original result for Tendermint (needed when joining clients need the response)
	var originalResult json.RawMessage
	if firstMsg != nil && firstMsg.Result != nil {
		originalResult = firstMsg.Result
	}

	// Create first reply - preserves original format for Tendermint, uses router ID for EVM
	firstReplyData, err := createSubscriptionReply(routerID, firstMsg, dwsm.apiInterface)
	if err != nil {
		upstreamSub.Unsubscribe()
		dwsm.failPendingSubscription(hashedParams)
		go dwsm.metricsManager.SetFailedWsSubscriptionRequestMetric(dwsm.chainID, dwsm.apiInterface)
		return nil, nil, fmt.Errorf("failed to create subscription reply: %w", err)
	}

	firstReply = &pairingtypes.RelayReply{
		Data: firstReplyData,
	}

	// Create subscription context
	subCtx, cancel := context.WithCancel(ctx)

	// Increment subscription count on the connection (for pool auto-scaling)
	conn.IncrementSubscriptions()

	// Store active subscription
	activeSub := &directActiveSubscription{
		upstreamPool:         pool,
		upstreamConnection:   conn, // Track which connection this subscription uses
		upstreamSubscription: upstreamSub,
		upstreamID:           upstreamID,
		clientRouterIDs:      make(map[string]string),
		hashedParams:         hashedParams,
		subscriptionParams:   subscriptionParams, // Store for restoration after reconnect
		subscribeMethod:      subscribeMethod,    // Store for restoration (eth_subscribe, subscribe, etc.)
		connectedClients:     make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		firstReply:           firstReply,
		originalResult:       originalResult, // Store for Tendermint joining clients
		ctx:                  subCtx,
		cancel:               cancel,
		closeSubChan:         make(chan struct{}),
		messagesChan:         msgChan, // Use the channel from upstream subscription
	}
	activeSub.connectedClients[clientKey] = safeChannelSender
	activeSub.clientRouterIDs[clientKey] = routerID

	dwsm.lock.Lock()
	dwsm.activeSubscriptions[hashedParams] = activeSub
	if dwsm.connectedClients[clientKey] == nil {
		dwsm.connectedClients[clientKey] = make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply])
	}
	dwsm.connectedClients[clientKey][hashedParams] = safeChannelSender
	dwsm.lock.Unlock()

	// Increment global subscription counter
	dwsm.totalSubscriptions.Add(1)

	// Notify pending subscriptions of success
	dwsm.successPendingSubscription(hashedParams)

	// Start listening for upstream messages
	go dwsm.listenForUpstreamMessages(subCtx, hashedParams, activeSub, upstreamSub)

	// Handle client disconnect
	go dwsm.handleClientDisconnect(ctx, clientKey, hashedParams)

	utils.LavaFormatInfo("DirectWS: subscription started",
		utils.LogAttr("routerID", routerID),
		utils.LogAttr("upstreamID", upstreamID),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	return firstReply, repliesChannel, nil
}

// Unsubscribe implements chainlib.WSSubscriptionManager.
// It handles an explicit unsubscribe request from a client.
func (dwsm *DirectWSSubscriptionManager) Unsubscribe(
	ctx context.Context,
	protocolMessage chainlib.ProtocolMessage,
	dappID string,
	consumerIp string,
	webSocketConnectionUniqueId string,
	metricsData *metrics.RelayMetrics,
) error {
	clientKey := dwsm.CreateWebSocketConnectionUniqueKey(dappID, consumerIp, webSocketConnectionUniqueId)

	// Check unsubscribe rate limiting
	if !dwsm.rateLimiter.AllowUnsubscribe(clientKey) {
		utils.LavaFormatWarning("DirectWS: client unsubscribe rate limit exceeded", nil,
			utils.LogAttr("clientKey", clientKey),
			utils.LogAttr("limit", dwsm.config.UnsubscribesPerMinutePerClient),
		)
		return fmt.Errorf("unsubscribe rate limit exceeded: max %d unsubscribes per minute",
			dwsm.config.UnsubscribesPerMinutePerClient)
	}

	// Extract subscription ID from the unsubscribe message
	// - EVM: Router ID (hex string we generated)
	// - Tendermint: Query string (the actual query used to subscribe)
	subIDFromRequest, err := dwsm.extractSubscriptionIDFromUnsubscribe(protocolMessage)
	if err != nil {
		return fmt.Errorf("failed to extract subscription ID: %w", err)
	}

	utils.LavaFormatTrace("DirectWS: unsubscribe request",
		utils.LogAttr("subIDFromRequest", subIDFromRequest),
		utils.LogAttr("clientKey", clientKey),
		utils.LogAttr("apiInterface", dwsm.apiInterface),
	)

	// SECURITY: Validate that the subscription belongs to the calling client BEFORE removing it.
	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	// Find the subscription this client is connected to and verify ownership
	var activeSub *directActiveSubscription
	var hashedParams string
	var routerSubID string // The router ID for this client (needed for ID mapper cleanup)
	var clientOwnsSubscription bool

	for hp, sub := range dwsm.activeSubscriptions {
		if ownedRouterID, exists := sub.clientRouterIDs[clientKey]; exists {
			// For Tendermint: client unsubscribes using the query (which is also the upstream ID)
			// For EVM: client unsubscribes using the router ID we gave them
			if dwsm.apiInterface == "tendermintrpc" {
				// Tendermint: match by upstream ID (query string)
				if sub.upstreamID == subIDFromRequest {
					activeSub = sub
					hashedParams = hp
					routerSubID = ownedRouterID
					clientOwnsSubscription = true
					break
				}
			} else {
				// EVM: match by router ID
				if ownedRouterID == subIDFromRequest {
					activeSub = sub
					hashedParams = hp
					routerSubID = ownedRouterID
					clientOwnsSubscription = true
					break
				}
			}
		}
	}

	if !clientOwnsSubscription {
		// Either the subscription doesn't exist or the client is trying to unsubscribe
		// from a subscription they don't own
		utils.LavaFormatWarning("DirectWS: unsubscribe rejected - client does not own subscription", nil,
			utils.LogAttr("subIDFromRequest", subIDFromRequest),
			utils.LogAttr("clientKey", clientKey),
		)
		return common.SubscriptionNotFoundError
	}

	if activeSub == nil {
		return common.SubscriptionNotFoundError
	}

	// Now that ownership is verified, remove the mapping
	_, lastClient := dwsm.idMapper.RemoveMapping(routerSubID)

	// Remove this client and their router ID
	if sender, ok := activeSub.connectedClients[clientKey]; ok {
		sender.Close()
		delete(activeSub.connectedClients, clientKey)
	}
	delete(activeSub.clientRouterIDs, clientKey)

	if clientSubs, ok := dwsm.connectedClients[clientKey]; ok {
		delete(clientSubs, hashedParams)
		if len(clientSubs) == 0 {
			delete(dwsm.connectedClients, clientKey)
		}
	}

	if lastClient {
		// This was the last client, unsubscribe upstream
		activeSub.upstreamSubscription.Unsubscribe()
		activeSub.cancel()
		delete(dwsm.activeSubscriptions, hashedParams)
		dwsm.totalSubscriptions.Add(-1) // Decrement global counter
		// Notify pool to potentially scale down
		if activeSub.upstreamConnection != nil {
			activeSub.upstreamPool.NotifySubscriptionRemoved(activeSub.upstreamConnection)
		}
		go dwsm.metricsManager.SetWsSubscriptioDisconnectRequestMetric(dwsm.chainID, dwsm.apiInterface, metrics.WsDisconnectionReasonUser)
	}

	return nil
}

// UnsubscribeAll implements chainlib.WSSubscriptionManager.
// It removes all subscriptions for a specific client connection.
func (dwsm *DirectWSSubscriptionManager) UnsubscribeAll(
	ctx context.Context,
	dappID string,
	consumerIp string,
	webSocketConnectionUniqueId string,
	metricsData *metrics.RelayMetrics,
) error {
	clientKey := dwsm.CreateWebSocketConnectionUniqueKey(dappID, consumerIp, webSocketConnectionUniqueId)

	// Note: UnsubscribeAll is NOT rate limited because:
	// 1. It's a cleanup/teardown operation called when a WebSocket connection closes
	// 2. Rate limiting could leave orphaned subscriptions consuming resources
	// 3. It's a single batch operation, not a repeated action like individual unsubscribes
	// The rate limiter's AllowUnsubscribe is designed for individual eth_unsubscribe calls.

	utils.LavaFormatTrace("DirectWS: unsubscribe all request",
		utils.LogAttr("clientKey", clientKey),
	)

	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	clientSubs, ok := dwsm.connectedClients[clientKey]
	if !ok {
		return nil // No subscriptions for this client
	}

	// Iterate over all subscriptions for this client
	for hashedParams := range clientSubs {
		activeSub, exists := dwsm.activeSubscriptions[hashedParams]
		if !exists {
			continue
		}

		// Get this client's router ID before removing
		clientRouterID := activeSub.clientRouterIDs[clientKey]

		// Close the client's channel
		if sender, ok := activeSub.connectedClients[clientKey]; ok {
			sender.Close()
			delete(activeSub.connectedClients, clientKey)
		}

		// Remove client's router ID from subscription
		delete(activeSub.clientRouterIDs, clientKey)

		// Remove this client's router ID mapping
		if clientRouterID != "" {
			dwsm.idMapper.RemoveMapping(clientRouterID)
		}

		// If no more clients, close upstream subscription
		if len(activeSub.connectedClients) == 0 {
			activeSub.upstreamSubscription.Unsubscribe()
			activeSub.cancel()
			delete(dwsm.activeSubscriptions, hashedParams)
			dwsm.totalSubscriptions.Add(-1) // Decrement global counter
			// Notify pool to potentially scale down
			if activeSub.upstreamConnection != nil {
				activeSub.upstreamPool.NotifySubscriptionRemoved(activeSub.upstreamConnection)
			}
			go dwsm.metricsManager.SetWsSubscriptioDisconnectRequestMetric(dwsm.chainID, dwsm.apiInterface, metrics.WsDisconnectionReasonUser)
		}
	}

	delete(dwsm.connectedClients, clientKey)

	// Clean up sticky session for this client
	dwsm.stickyStore.Delete(clientKey)

	// Clean up rate limiter for this client
	dwsm.rateLimiter.CleanupClient(clientKey)

	utils.LavaFormatTrace("DirectWS: cleared sticky session and rate limiter on unsubscribe all",
		utils.LogAttr("clientKey", clientKey),
	)

	return nil
}

// getHashedParams extracts and hashes the subscription parameters from a protocol message
func (dwsm *DirectWSSubscriptionManager) getHashedParams(protocolMessage chainlib.ProtocolMessage) (hashedParams string, params []byte, err error) {
	params, err = gojson.Marshal(protocolMessage.GetRPCMessage().GetParams())
	if err != nil {
		return "", nil, utils.LavaFormatError("could not marshal params", err)
	}

	hashedParams = rpcclient.CreateHashFromParams(params)
	return hashedParams, params, nil
}

// extractSubscriptionIDFromUnsubscribe extracts the subscription ID from an unsubscribe message
// Handles both EVM and Tendermint formats:
// - EVM (eth_unsubscribe): params is ["0x..."] (array with hex subscription ID)
// - Tendermint (unsubscribe): params is {"query": "tm.event='NewBlock'"} (object with query field)
func (dwsm *DirectWSSubscriptionManager) extractSubscriptionIDFromUnsubscribe(protocolMessage chainlib.ProtocolMessage) (string, error) {
	params := protocolMessage.GetRPCMessage().GetParams()
	if params == nil {
		return "", fmt.Errorf("no params in unsubscribe message")
	}

	paramsBytes, err := gojson.Marshal(params)
	if err != nil {
		return "", err
	}

	// For Tendermint: try to extract from object with "query" field
	// Format: {"query": "tm.event='NewBlock'"}
	if dwsm.apiInterface == "tendermintrpc" {
		var paramsMap map[string]any
		if err := gojson.Unmarshal(paramsBytes, &paramsMap); err == nil {
			if query, ok := paramsMap["query"].(string); ok && query != "" {
				return query, nil
			}
		}
	}

	// For EVM: try to extract as array (JSON-RPC style: ["subscription_id"])
	var paramsArray []any
	if err := gojson.Unmarshal(paramsBytes, &paramsArray); err == nil && len(paramsArray) > 0 {
		if id, ok := paramsArray[0].(string); ok {
			return id, nil
		}
	}

	// Try as string directly
	if id, ok := params.(string); ok {
		return id, nil
	}

	return "", fmt.Errorf("could not extract subscription ID from params: %v", params)
}

// checkForActiveSubscriptionAndConnect checks if there's an active subscription and connects the client.
// If joining an existing subscription, generates a unique router ID for this client and registers it
// with the ID mapper. This ensures each client has their own subscription ID that maps to the
// shared upstream subscription, enabling proper unsubscribe behavior with deduplication.
// The requestID parameter is the client's original JSON-RPC request ID, which must be included
// in the response per JSON-RPC 2.0 spec.
func (dwsm *DirectWSSubscriptionManager) checkForActiveSubscriptionAndConnect(
	ctx context.Context,
	hashedParams string,
	clientKey string,
	safeChannelSender *common.SafeChannelSender[*pairingtypes.RelayReply],
	requestID json.RawMessage,
) (*pairingtypes.RelayReply, bool) {
	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	activeSub, found := dwsm.activeSubscriptions[hashedParams]
	if !found {
		return nil, false
	}

	// Check if client is already connected - return their existing router ID's reply
	if existingRouterID, exists := activeSub.clientRouterIDs[clientKey]; exists {
		// Create reply with client's existing router ID and their request ID
		// For Tendermint: uses originalResult to preserve {"query":"..."} format
		replyData, err := createSubscriptionReplyFromRouterID(existingRouterID, requestID, dwsm.apiInterface, activeSub.originalResult)
		if err != nil {
			utils.LavaFormatWarning("DirectWS: failed to create reply for existing client", err)
			return activeSub.firstReply, false
		}
		return &pairingtypes.RelayReply{Data: replyData}, false
	}

	// Generate a unique router ID for this joining client (for EVM deduplication)
	// For Tendermint, we still generate IDs for internal tracking but don't expose them
	clientRouterID := dwsm.idMapper.GenerateRouterID(clientKey)

	// Register the mapping: this client's router ID -> shared upstream ID
	dwsm.idMapper.RegisterMapping(clientRouterID, activeSub.upstreamID)

	// Add client to existing subscription with their unique router ID
	activeSub.connectedClients[clientKey] = safeChannelSender
	activeSub.clientRouterIDs[clientKey] = clientRouterID
	if dwsm.connectedClients[clientKey] == nil {
		dwsm.connectedClients[clientKey] = make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply])
	}
	dwsm.connectedClients[clientKey][hashedParams] = safeChannelSender

	utils.LavaFormatTrace("DirectWS: client joined existing subscription with unique router ID",
		utils.LogAttr("clientKey", clientKey),
		utils.LogAttr("clientRouterID", clientRouterID),
		utils.LogAttr("upstreamID", activeSub.upstreamID),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	// Create first reply with this client's unique router ID and their request ID
	// For Tendermint: uses originalResult to preserve {"query":"..."} format
	replyData, err := createSubscriptionReplyFromRouterID(clientRouterID, requestID, dwsm.apiInterface, activeSub.originalResult)
	if err != nil {
		utils.LavaFormatWarning("DirectWS: failed to create reply for joining client", err)
		return activeSub.firstReply, true // Fallback to original reply
	}

	return &pairingtypes.RelayReply{Data: replyData}, true // Joined existing subscription
}

// checkAndAddPendingSubscription checks for pending subscriptions
func (dwsm *DirectWSSubscriptionManager) checkAndAddPendingSubscription(hashedParams string) (chan bool, bool) {
	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	pending, found := dwsm.pendingSubscriptions[hashedParams]
	if !found {
		// No pending, we are the first - create pending entry
		dwsm.pendingSubscriptions[hashedParams] = &pendingSubscriptionsBroadcastManager{}
		return nil, false
	}

	// There's a pending subscription, wait for it
	listenChan := make(chan bool, 1)
	pending.broadcastChannelList = append(pending.broadcastChannelList, listenChan)
	return listenChan, true
}

// failPendingSubscription notifies waiting clients that subscription failed
func (dwsm *DirectWSSubscriptionManager) failPendingSubscription(hashedParams string) {
	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	pending, found := dwsm.pendingSubscriptions[hashedParams]
	if found {
		pending.broadcastToChannelList(false)
		delete(dwsm.pendingSubscriptions, hashedParams)
	}
}

// successPendingSubscription notifies waiting clients that subscription succeeded
func (dwsm *DirectWSSubscriptionManager) successPendingSubscription(hashedParams string) {
	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	pending, found := dwsm.pendingSubscriptions[hashedParams]
	if found {
		pending.broadcastToChannelList(true)
		delete(dwsm.pendingSubscriptions, hashedParams)
	}
}

// createUpstreamSubscription creates a subscription on the upstream WebSocket
// Returns the subscription, first message, message channel, and error
// The subscribeMethod parameter allows using different subscription methods for different protocols:
// - EVM chains: "eth_subscribe"
// - Tendermint: "subscribe"
func (dwsm *DirectWSSubscriptionManager) createUpstreamSubscription(
	ctx context.Context,
	conn *UpstreamWSConnection,
	params []byte,
	subscribeMethod string,
) (*rpcclient.ClientSubscription, *rpcclient.JsonrpcMessage, chan *rpcclient.JsonrpcMessage, error) {
	client := conn.GetClient()
	if client == nil {
		return nil, nil, nil, fmt.Errorf("no WebSocket client available")
	}

	// Create channel for subscription messages - this is where upstream will deliver notifications
	msgChan := make(chan *rpcclient.JsonrpcMessage, 100)

	// Parse params back to interface for the Subscribe call
	var subscriptionParams interface{}
	if err := json.Unmarshal(params, &subscriptionParams); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	utils.LavaFormatTrace("DirectWS: creating upstream subscription",
		utils.LogAttr("method", subscribeMethod),
		utils.LogAttr("apiInterface", dwsm.apiInterface),
	)

	// Subscribe using rpcclient with the appropriate method for the protocol
	sub, firstMsg, err := client.Subscribe(ctx, nil, subscribeMethod, msgChan, subscriptionParams)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("upstream subscribe failed: %w", err)
	}

	return sub, firstMsg, msgChan, nil
}

// listenForUpstreamMessages listens for messages from upstream and routes to clients
func (dwsm *DirectWSSubscriptionManager) listenForUpstreamMessages(
	ctx context.Context,
	hashedParams string,
	activeSub *directActiveSubscription,
	upstreamSub *rpcclient.ClientSubscription,
) {
	defer func() {
		dwsm.cleanupSubscription(hashedParams)
	}()

	for {
		select {
		case <-ctx.Done():
			utils.LavaFormatTrace("DirectWS: subscription context done",
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
			return

		case <-activeSub.closeSubChan:
			utils.LavaFormatTrace("DirectWS: subscription close requested",
				utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
			)
			return

		case err := <-upstreamSub.Err():
			if err != nil {
				utils.LavaFormatWarning("DirectWS: upstream subscription error",
					err,
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				)
				// Attempt reconnection
				go dwsm.handleUpstreamDisconnect(ctx, hashedParams, activeSub)
			}
			return

		case msg := <-activeSub.messagesChan:
			if msg == nil {
				continue
			}
			dwsm.routeMessageToClients(hashedParams, activeSub, msg)
		}
	}
}

// routeMessageToClients routes an upstream message to all connected clients.
// Each client receives the message with their unique subscription ID (router ID)
// rewritten in the notification params. This enables proper deduplication where
// clients can unsubscribe individually without affecting other clients.
func (dwsm *DirectWSSubscriptionManager) routeMessageToClients(
	hashedParams string,
	activeSub *directActiveSubscription,
	msg *rpcclient.JsonrpcMessage,
) {
	dwsm.lock.RLock()
	// Copy client channels and their router IDs
	type clientInfo struct {
		sender   *common.SafeChannelSender[*pairingtypes.RelayReply]
		routerID string
	}
	clients := make(map[string]clientInfo)
	for clientKey, sender := range activeSub.connectedClients {
		routerID := activeSub.clientRouterIDs[clientKey]
		clients[clientKey] = clientInfo{sender: sender, routerID: routerID}
	}
	dwsm.lock.RUnlock()

	// Rewrite subscription ID per-client and send
	for clientKey, info := range clients {
		// Rewrite the message with this client's unique router ID
		rewrittenMsg, err := rewriteSubscriptionID(msg, info.routerID)
		if err != nil {
			utils.LavaFormatWarning("DirectWS: failed to rewrite subscription ID for client", err,
				utils.LogAttr("clientKey", clientKey),
			)
			continue
		}

		reply := &pairingtypes.RelayReply{
			Data: rewrittenMsg,
		}

		info.sender.Send(reply)
		utils.LavaFormatTrace("DirectWS: sent message to client with unique router ID",
			utils.LogAttr("clientKey", clientKey),
			utils.LogAttr("routerID", info.routerID),
		)
	}
}

// handleUpstreamDisconnect handles upstream connection loss and attempts to restore subscriptions
func (dwsm *DirectWSSubscriptionManager) handleUpstreamDisconnect(
	ctx context.Context,
	hashedParams string,
	activeSub *directActiveSubscription,
) {
	// Prevent concurrent restoration attempts
	if !activeSub.restoring.CompareAndSwap(false, true) {
		utils.LavaFormatDebug("DirectWS: restoration already in progress",
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
		return
	}
	defer activeSub.restoring.Store(false)

	utils.LavaFormatInfo("DirectWS: handling upstream disconnect",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("connectedClients", len(activeSub.clientRouterIDs)),
	)

	// Try to reconnect the WebSocket connection
	err := activeSub.upstreamPool.ReconnectWithBackoff(ctx)
	if err != nil {
		utils.LavaFormatError("DirectWS: reconnection failed, cleaning up subscription", err,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
		dwsm.cleanupSubscription(hashedParams)
		return
	}

	// Connection restored, now restore the subscription
	utils.LavaFormatInfo("DirectWS: connection restored, restoring subscription",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	// Get the restored connection
	conn, err := activeSub.upstreamPool.GetConnection(ctx)
	if err != nil {
		utils.LavaFormatError("DirectWS: failed to get connection after reconnect", err,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
		dwsm.cleanupSubscription(hashedParams)
		return
	}

	// Re-subscribe using the stored params and method
	newUpstreamSub, firstMsg, newMsgChan, err := dwsm.createUpstreamSubscription(ctx, conn, activeSub.subscriptionParams, activeSub.subscribeMethod)
	if err != nil {
		utils.LavaFormatError("DirectWS: failed to restore subscription", err,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
		dwsm.cleanupSubscription(hashedParams)
		return
	}

	// Extract new upstream subscription ID based on API interface
	newUpstreamID := getSubscriptionID(dwsm.apiInterface, firstMsg, activeSub.subscriptionParams)

	// Increment subscription count on the NEW connection (matching initial creation behavior)
	conn.IncrementSubscriptions()

	// Update the ID mapping, message channel, and connection bookkeeping
	dwsm.lock.Lock()
	oldUpstreamID := activeSub.upstreamID
	oldConnection := activeSub.upstreamConnection // Keep reference to old connection for cleanup

	activeSub.upstreamSubscription = newUpstreamSub
	activeSub.upstreamID = newUpstreamID
	activeSub.upstreamConnection = conn    // Update to new connection
	activeSub.messagesChan = newMsgChan    // Update to new channel from upstream

	// Collect all client router IDs to re-register
	clientRouterIDs := make([]string, 0, len(activeSub.clientRouterIDs))
	for _, routerID := range activeSub.clientRouterIDs {
		clientRouterIDs = append(clientRouterIDs, routerID)
	}
	dwsm.lock.Unlock()

	// Notify the pool that the OLD connection no longer has this subscription
	// This decrements the subscription count on the stale connection
	if oldConnection != nil && oldConnection != conn {
		activeSub.upstreamPool.NotifySubscriptionRemoved(oldConnection)
	}

	// Update the ID mapper - remove old mappings and re-register all client router IDs
	// to the new upstream ID
	dwsm.idMapper.RemoveAllForUpstream(oldUpstreamID)
	for _, routerID := range clientRouterIDs {
		dwsm.idMapper.RegisterMapping(routerID, newUpstreamID)
	}

	utils.LavaFormatInfo("DirectWS: subscription restored successfully",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("oldUpstreamID", oldUpstreamID),
		utils.LogAttr("newUpstreamID", newUpstreamID),
		utils.LogAttr("connectedClients", len(clientRouterIDs)),
	)

	// Start listening for messages on the new subscription
	go dwsm.listenForUpstreamMessages(activeSub.ctx, hashedParams, activeSub, newUpstreamSub)
}

// handleClientDisconnect handles client WebSocket disconnection.
// It removes the client's router ID mapping and cleans up the subscription if this was the last client.
func (dwsm *DirectWSSubscriptionManager) handleClientDisconnect(
	ctx context.Context,
	clientKey string,
	hashedParams string,
) {
	<-ctx.Done()

	utils.LavaFormatTrace("DirectWS: client disconnected",
		utils.LogAttr("clientKey", clientKey),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	// Remove client from subscription
	activeSub, found := dwsm.activeSubscriptions[hashedParams]
	if !found {
		return
	}

	// Get the client's router ID before removing
	clientRouterID := activeSub.clientRouterIDs[clientKey]

	// Remove client from connected clients and router IDs
	delete(activeSub.connectedClients, clientKey)
	delete(activeSub.clientRouterIDs, clientKey)

	// Remove the client's router ID mapping from the ID mapper
	if clientRouterID != "" {
		dwsm.idMapper.RemoveMapping(clientRouterID)
	}

	if clientSubs, ok := dwsm.connectedClients[clientKey]; ok {
		delete(clientSubs, hashedParams)
		if len(clientSubs) == 0 {
			delete(dwsm.connectedClients, clientKey)
			// Clean up sticky session and rate limiter when client has no more subscriptions
			dwsm.stickyStore.Delete(clientKey)
			dwsm.rateLimiter.CleanupClient(clientKey)
			utils.LavaFormatTrace("DirectWS: cleared sticky session and rate limiter on client disconnect",
				utils.LogAttr("clientKey", clientKey),
			)
		}
	}

	// If no more clients, close the upstream subscription
	if len(activeSub.connectedClients) == 0 {
		utils.LavaFormatTrace("DirectWS: no more clients, closing upstream subscription",
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
		// Capture connection reference before async cleanup
		upstreamConn := activeSub.upstreamConnection
		upstreamPool := activeSub.upstreamPool
		go func() {
			activeSub.upstreamSubscription.Unsubscribe()
			activeSub.cancel()
			close(activeSub.closeSubChan)
		}()
		delete(dwsm.activeSubscriptions, hashedParams)
		dwsm.totalSubscriptions.Add(-1) // Decrement global counter
		// Notify pool to potentially scale down
		if upstreamConn != nil {
			upstreamPool.NotifySubscriptionRemoved(upstreamConn)
		}
		go dwsm.metricsManager.SetWsSubscriptioDisconnectRequestMetric(dwsm.chainID, dwsm.apiInterface, metrics.WsDisconnectionReasonConsumer)
	}
}

// cleanupSubscription cleans up a subscription
func (dwsm *DirectWSSubscriptionManager) cleanupSubscription(hashedParams string) {
	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	activeSub, found := dwsm.activeSubscriptions[hashedParams]
	if !found {
		return
	}

	// Close all client channels
	for clientKey, sender := range activeSub.connectedClients {
		sender.Close()
		if clientSubs, ok := dwsm.connectedClients[clientKey]; ok {
			delete(clientSubs, hashedParams)
			if len(clientSubs) == 0 {
				delete(dwsm.connectedClients, clientKey)
			}
		}
	}

	// Remove ID mappings
	dwsm.idMapper.RemoveAllForUpstream(activeSub.upstreamID)

	// Cleanup subscription
	activeSub.cancel()
	delete(dwsm.activeSubscriptions, hashedParams)
	dwsm.totalSubscriptions.Add(-1) // Decrement global counter

	// Notify pool to potentially scale down
	if activeSub.upstreamConnection != nil {
		activeSub.upstreamPool.NotifySubscriptionRemoved(activeSub.upstreamConnection)
	}

	go dwsm.metricsManager.SetWsSubscriptioDisconnectRequestMetric(dwsm.chainID, dwsm.apiInterface, metrics.WsDisconnectionReasonProvider)

	utils.LavaFormatTrace("DirectWS: subscription cleaned up",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)
}

// Close closes the subscription manager and all connections
func (dwsm *DirectWSSubscriptionManager) Close() {
	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	// Close all subscriptions
	for _, sub := range dwsm.activeSubscriptions {
		sub.upstreamSubscription.Unsubscribe()
		sub.cancel()
		for _, sender := range sub.connectedClients {
			sender.Close()
		}
	}
	dwsm.activeSubscriptions = make(map[string]*directActiveSubscription)

	// Close all pools
	for _, pool := range dwsm.upstreamPools {
		pool.Close()
	}
	dwsm.upstreamPools = make(map[string]*UpstreamWSPool)

	utils.LavaFormatInfo("DirectWS: subscription manager closed")
}

// Helper functions

// extractRequestID extracts the JSON-RPC request ID from raw request data.
// This is needed because GenericMessage interface doesn't expose GetID().
// Returns the raw JSON representation of the ID to preserve its type (string, number, null).
func extractRequestID(data []byte) json.RawMessage {
	if len(data) == 0 {
		return json.RawMessage("1") // Default fallback
	}

	var req struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(data, &req); err != nil || req.ID == nil {
		return json.RawMessage("1") // Default fallback
	}
	return req.ID
}

// extractSubscriptionID extracts the subscription ID from the first response (EVM style)
// For EVM chains: the subscription ID is returned as a hex string in the result field
func extractSubscriptionID(msg *rpcclient.JsonrpcMessage) string {
	if msg == nil || msg.Result == nil {
		return ""
	}
	var id string
	if err := json.Unmarshal(msg.Result, &id); err != nil {
		return ""
	}
	return id
}

// extractTendermintSubscriptionID extracts the subscription ID from Tendermint request params
// For Tendermint: the subscription ID is the "query" parameter from the request
// Example: {"query": "tm.event='NewBlock'"} -> subscription ID is "tm.event='NewBlock'"
func extractTendermintSubscriptionID(params []byte) string {
	if params == nil {
		return ""
	}
	var paramsMap map[string]interface{}
	if err := json.Unmarshal(params, &paramsMap); err != nil {
		return ""
	}
	if query, ok := paramsMap["query"].(string); ok {
		return query
	}
	return ""
}

// getSubscriptionID returns the appropriate subscription ID based on the protocol
// For EVM chains: extracts from response result (hex string)
// For Tendermint: extracts from request params (query field)
func getSubscriptionID(apiInterface string, responseMsg *rpcclient.JsonrpcMessage, requestParams []byte) string {
	if apiInterface == "tendermintrpc" {
		return extractTendermintSubscriptionID(requestParams)
	}
	return extractSubscriptionID(responseMsg)
}

// createSubscriptionReply creates the first reply with the router subscription ID
func createSubscriptionReply(routerID string, originalMsg *rpcclient.JsonrpcMessage, apiInterface string) ([]byte, error) {
	if originalMsg == nil {
		return nil, fmt.Errorf("original message is nil")
	}

	// For Tendermint: preserve the original response format {"result":{"query":"..."}}
	// Tendermint clients expect the query object back, not a router ID
	if apiInterface == "tendermintrpc" {
		// Return original message as-is - it already has the correct format
		return json.Marshal(originalMsg)
	}

	// For EVM: create response with router ID instead of upstream ID
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      originalMsg.ID,
		"result":  routerID,
	}

	return json.Marshal(response)
}

// createSubscriptionReplyFromRouterID creates a subscription reply with the router ID and matching request ID.
// Used when a client joins an existing subscription and needs their unique router ID in the response.
// The requestID must match the client's original eth_subscribe request per JSON-RPC 2.0 spec.
// For Tendermint, originalResult contains the query object that must be preserved.
func createSubscriptionReplyFromRouterID(routerID string, requestID json.RawMessage, apiInterface string, originalResult json.RawMessage) ([]byte, error) {
	// For Tendermint: preserve the original result format {"query":"..."}
	// Tendermint clients expect the query object back, not a router ID
	if apiInterface == "tendermintrpc" && originalResult != nil {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(requestID),
			"result":  json.RawMessage(originalResult), // Preserve {"query":"..."} format
		}
		return json.Marshal(response)
	}

	// For EVM: create response with the client's unique router ID and their original request ID
	// JSON-RPC 2.0 requires the response ID to match the request ID exactly
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      json.RawMessage(requestID), // Use client's actual request ID
		"result":  routerID,
	}

	return json.Marshal(response)
}

// rewriteSubscriptionID rewrites the subscription ID in a notification message
// For EVM (eth_subscription): rewrites params.subscription with router ID
// For Tendermint: notifications contain query in result, passed through as-is
func rewriteSubscriptionID(msg *rpcclient.JsonrpcMessage, routerID string) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	// EVM subscription notifications format:
	// {"jsonrpc":"2.0","method":"eth_subscription","params":{"subscription":"0x...","result":{...}}}
	if msg.Method == "eth_subscription" && msg.Params != nil {
		var params struct {
			Subscription string          `json:"subscription"`
			Result       json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return nil, err
		}

		// Rewrite with router ID
		newParams := map[string]any{
			"subscription": routerID,
			"result":       params.Result,
		}
		paramsBytes, err := json.Marshal(newParams)
		if err != nil {
			return nil, err
		}

		response := map[string]any{
			"jsonrpc": "2.0",
			"method":  "eth_subscription",
			"params":  json.RawMessage(paramsBytes),
		}
		return json.Marshal(response)
	}

	// Tendermint subscription notifications format:
	// {"jsonrpc":"2.0","id":null,"result":{"query":"tm.event='NewBlock'","data":{...}}}
	// The query field identifies the subscription - we pass through as-is since:
	// 1. Clients expect to see their actual query in notifications
	// 2. The router ID is only used for unsubscribe operations
	if msg.Result != nil {
		var result struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(msg.Result, &result); err == nil && result.Query != "" {
			// This is a Tendermint notification - pass through unchanged
			return json.Marshal(msg)
		}
	}

	// For other messages, return as-is
	return json.Marshal(msg)
}

// Compile-time interface compliance check
var _ chainlib.WSSubscriptionManager = (*DirectWSSubscriptionManager)(nil)
