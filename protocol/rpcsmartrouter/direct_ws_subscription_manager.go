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

	// Router-generated info
	routerID     string // ID returned to clients
	hashedParams string // Hash of subscription parameters

	// Subscription params for restoration after reconnect
	subscriptionParams []byte // Original subscription params (JSON)

	// Client tracking - multiple clients can share one upstream subscription
	connectedClients map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]

	// First reply cached for deduplication
	firstReply *pairingtypes.RelayReply

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

	clientKey := dwsm.CreateWebSocketConnectionUniqueKey(dappID, consumerIp, webSocketConnectionUniqueId)

	utils.LavaFormatTrace("DirectWS: request to start subscription",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("clientKey", clientKey),
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
	existingReply, joined := dwsm.checkForActiveSubscriptionAndConnect(ctx, hashedParams, clientKey, safeChannelSender)
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
				existingReply, joined := dwsm.checkForActiveSubscriptionAndConnect(ctx, hashedParams, clientKey, safeChannelSender)
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
	upstreamSub, firstMsg, msgChan, err := dwsm.createUpstreamSubscription(ctx, conn, subscriptionParams)
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

	// Generate router subscription ID
	routerID := dwsm.idMapper.GenerateRouterID(clientKey)

	// Extract upstream subscription ID from first message
	upstreamID := extractSubscriptionID(firstMsg)
	dwsm.idMapper.RegisterMapping(routerID, upstreamID)

	// Create first reply with router ID
	firstReplyData, err := createSubscriptionReply(routerID, firstMsg)
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
		routerID:             routerID,
		hashedParams:         hashedParams,
		subscriptionParams:   subscriptionParams, // Store for restoration after reconnect
		connectedClients:     make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		firstReply:           firstReply,
		ctx:                  subCtx,
		cancel:               cancel,
		closeSubChan:         make(chan struct{}),
		messagesChan:         msgChan, // Use the channel from upstream subscription
	}
	activeSub.connectedClients[clientKey] = safeChannelSender

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

	// Extract subscription ID from the unsubscribe message
	// The params contain the subscription ID to unsubscribe from
	routerSubID, err := dwsm.extractSubscriptionIDFromUnsubscribe(protocolMessage)
	if err != nil {
		return fmt.Errorf("failed to extract subscription ID: %w", err)
	}

	utils.LavaFormatTrace("DirectWS: unsubscribe request",
		utils.LogAttr("routerSubID", routerSubID),
		utils.LogAttr("clientKey", clientKey),
	)

	upstreamID, lastClient := dwsm.idMapper.RemoveMapping(routerSubID)
	if upstreamID == "" {
		return common.SubscriptionNotFoundError
	}

	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	// Find the active subscription by upstream ID
	var activeSub *directActiveSubscription
	var hashedParams string
	for hp, sub := range dwsm.activeSubscriptions {
		if sub.upstreamID == upstreamID {
			activeSub = sub
			hashedParams = hp
			break
		}
	}

	if activeSub == nil {
		return nil // Already cleaned up
	}

	// Remove this client
	if sender, ok := activeSub.connectedClients[clientKey]; ok {
		sender.Close()
		delete(activeSub.connectedClients, clientKey)
	}

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

		// Close the client's channel
		if sender, ok := activeSub.connectedClients[clientKey]; ok {
			sender.Close()
			delete(activeSub.connectedClients, clientKey)
		}

		// Remove router ID mapping
		dwsm.idMapper.RemoveMapping(activeSub.routerID)

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
func (dwsm *DirectWSSubscriptionManager) extractSubscriptionIDFromUnsubscribe(protocolMessage chainlib.ProtocolMessage) (string, error) {
	params := protocolMessage.GetRPCMessage().GetParams()
	if params == nil {
		return "", fmt.Errorf("no params in unsubscribe message")
	}

	// Try to extract as array (JSON-RPC style: ["subscription_id"])
	paramsBytes, err := gojson.Marshal(params)
	if err != nil {
		return "", err
	}

	var paramsArray []interface{}
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

// checkForActiveSubscriptionAndConnect checks if there's an active subscription and connects the client
func (dwsm *DirectWSSubscriptionManager) checkForActiveSubscriptionAndConnect(
	ctx context.Context,
	hashedParams string,
	clientKey string,
	safeChannelSender *common.SafeChannelSender[*pairingtypes.RelayReply],
) (*pairingtypes.RelayReply, bool) {
	dwsm.lock.Lock()
	defer dwsm.lock.Unlock()

	activeSub, found := dwsm.activeSubscriptions[hashedParams]
	if !found {
		return nil, false
	}

	// Check if client is already connected
	if _, exists := activeSub.connectedClients[clientKey]; exists {
		return activeSub.firstReply, false // Already connected, no new channel needed
	}

	// Add client to existing subscription
	activeSub.connectedClients[clientKey] = safeChannelSender
	if dwsm.connectedClients[clientKey] == nil {
		dwsm.connectedClients[clientKey] = make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply])
	}
	dwsm.connectedClients[clientKey][hashedParams] = safeChannelSender

	utils.LavaFormatTrace("DirectWS: client joined existing subscription",
		utils.LogAttr("clientKey", clientKey),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	return activeSub.firstReply, true // Joined existing subscription
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
func (dwsm *DirectWSSubscriptionManager) createUpstreamSubscription(
	ctx context.Context,
	conn *UpstreamWSConnection,
	params []byte,
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

	// Subscribe using rpcclient
	sub, firstMsg, err := client.Subscribe(ctx, nil, "eth_subscribe", msgChan, subscriptionParams)
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

// routeMessageToClients routes an upstream message to all connected clients
func (dwsm *DirectWSSubscriptionManager) routeMessageToClients(
	hashedParams string,
	activeSub *directActiveSubscription,
	msg *rpcclient.JsonrpcMessage,
) {
	dwsm.lock.RLock()
	clients := make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply])
	for k, v := range activeSub.connectedClients {
		clients[k] = v
	}
	dwsm.lock.RUnlock()

	// Rewrite subscription ID in the message
	rewrittenMsg, err := rewriteSubscriptionID(msg, activeSub.routerID)
	if err != nil {
		utils.LavaFormatWarning("DirectWS: failed to rewrite subscription ID", err)
		return
	}

	reply := &pairingtypes.RelayReply{
		Data: rewrittenMsg,
	}

	for clientKey, sender := range clients {
		sender.Send(reply)
		utils.LavaFormatTrace("DirectWS: sent message to client",
			utils.LogAttr("clientKey", clientKey),
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
		utils.LogAttr("routerID", activeSub.routerID),
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

	// Re-subscribe using the stored params
	newUpstreamSub, firstMsg, newMsgChan, err := dwsm.createUpstreamSubscription(ctx, conn, activeSub.subscriptionParams)
	if err != nil {
		utils.LavaFormatError("DirectWS: failed to restore subscription", err,
			utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		)
		dwsm.cleanupSubscription(hashedParams)
		return
	}

	// Extract new upstream subscription ID
	newUpstreamID := extractSubscriptionID(firstMsg)

	// Update the ID mapping and message channel
	dwsm.lock.Lock()
	oldUpstreamID := activeSub.upstreamID
	activeSub.upstreamSubscription = newUpstreamSub
	activeSub.upstreamID = newUpstreamID
	activeSub.messagesChan = newMsgChan // Update to new channel from upstream
	dwsm.lock.Unlock()

	// Update the ID mapper - remove old mapping and add new one
	dwsm.idMapper.RemoveAllForUpstream(oldUpstreamID)
	dwsm.idMapper.RegisterMapping(activeSub.routerID, newUpstreamID)

	utils.LavaFormatInfo("DirectWS: subscription restored successfully",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
		utils.LogAttr("routerID", activeSub.routerID),
		utils.LogAttr("oldUpstreamID", oldUpstreamID),
		utils.LogAttr("newUpstreamID", newUpstreamID),
		utils.LogAttr("connectedClients", len(activeSub.connectedClients)),
	)

	// Start listening for messages on the new subscription
	go dwsm.listenForUpstreamMessages(activeSub.ctx, hashedParams, activeSub, newUpstreamSub)
}

// handleClientDisconnect handles client WebSocket disconnection
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

	delete(activeSub.connectedClients, clientKey)
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

// extractSubscriptionID extracts the subscription ID from the first response
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

// createSubscriptionReply creates the first reply with the router subscription ID
func createSubscriptionReply(routerID string, originalMsg *rpcclient.JsonrpcMessage) ([]byte, error) {
	if originalMsg == nil {
		return nil, fmt.Errorf("original message is nil")
	}

	// Create response with router ID instead of upstream ID
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      originalMsg.ID,
		"result":  routerID,
	}

	return json.Marshal(response)
}

// rewriteSubscriptionID rewrites the subscription ID in a notification message
func rewriteSubscriptionID(msg *rpcclient.JsonrpcMessage, routerID string) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	// For subscription notifications, the format is:
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
		newParams := map[string]interface{}{
			"subscription": routerID,
			"result":       params.Result,
		}
		paramsBytes, err := json.Marshal(newParams)
		if err != nil {
			return nil, err
		}

		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "eth_subscription",
			"params":  json.RawMessage(paramsBytes),
		}
		return json.Marshal(response)
	}

	// For other messages, return as-is
	return json.Marshal(msg)
}

// Compile-time interface compliance check
var _ chainlib.WSSubscriptionManager = (*DirectWSSubscriptionManager)(nil)
