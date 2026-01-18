package rpcsmartrouter

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"google.golang.org/grpc"
)

// grpcActiveSubscription holds state for an active upstream gRPC stream
type grpcActiveSubscription struct {
	// Upstream connection info
	upstreamPool       *UpstreamGRPCPool
	upstreamConnection *UpstreamGRPCStreamConnection
	upstreamStream     grpc.ClientStream
	methodDescriptor   *desc.MethodDescriptor

	// Router-generated unique subscription ID
	routerSubscriptionID string
	hashedParams         string // Hash of method + request params

	// Client tracking - multiple clients can share ONE upstream stream
	clientRouterIDs  map[string]string                                        // clientKey -> routerID
	connectedClients map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]

	// Request info for restoration
	methodPath    string
	requestParams []byte

	// First reply cached for late joiners
	firstReply *pairingtypes.RelayReply

	// Lifecycle management
	ctx          context.Context
	cancel       context.CancelFunc
	closeSubChan chan struct{}

	// Restoration state
	restoring atomic.Bool

	// Message sequence counter
	messageSeq atomic.Uint64

	lock sync.RWMutex
}

// DirectGRPCSubscriptionManager manages gRPC streaming subscriptions directly to upstream endpoints.
// This follows the same pattern as DirectWSSubscriptionManager for consistency.
type DirectGRPCSubscriptionManager struct {
	// Active subscriptions keyed by hashedParams (method + request params)
	activeSubscriptions map[string]*grpcActiveSubscription

	// Pending subscriptions for deduplication (prevent duplicate upstream connections)
	pendingSubscriptions map[string]*pendingSubscriptionsBroadcastManager

	// Upstream connection pools keyed by endpoint URL
	upstreamPools map[string]*UpstreamGRPCPool

	// Subscription ID mapping (reuse from WS manager)
	idMapper *SubscriptionIDMapper

	// Dependencies
	metricsManager *metrics.ConsumerMetricsManager
	chainID        string
	apiInterface   string

	// Upstream gRPC endpoints
	grpcEndpoints  []*common.NodeUrl
	endpointsByURL map[string]*common.NodeUrl

	// Endpoint selection (can be nil)
	optimizer WebSocketEndpointOptimizer

	// Configuration
	config *GRPCStreamingConfig

	// Rate limiting
	rateLimiter *GRPCClientRateLimiter

	// Sticky sessions (client -> endpoint affinity)
	stickySessions map[string]string // clientKey -> endpoint URL

	// Total subscription counter
	totalSubscriptions atomic.Int64

	// Per-client subscription tracking
	clientSubscriptions map[string]map[string]struct{} // clientKey -> set of hashedParams

	// Manager state
	ctx    context.Context
	cancel context.CancelFunc

	lock sync.RWMutex
}

// NewDirectGRPCSubscriptionManager creates a new gRPC subscription manager
func NewDirectGRPCSubscriptionManager(
	metricsManager *metrics.ConsumerMetricsManager,
	chainID string,
	apiInterface string,
	grpcEndpoints []*common.NodeUrl,
	optimizer WebSocketEndpointOptimizer,
	config *GRPCStreamingConfig,
) *DirectGRPCSubscriptionManager {
	if config == nil {
		config = DefaultGRPCStreamingConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &DirectGRPCSubscriptionManager{
		activeSubscriptions:  make(map[string]*grpcActiveSubscription),
		pendingSubscriptions: make(map[string]*pendingSubscriptionsBroadcastManager),
		upstreamPools:        make(map[string]*UpstreamGRPCPool),
		idMapper:             NewSubscriptionIDMapper(),
		metricsManager:       metricsManager,
		chainID:              chainID,
		apiInterface:         apiInterface,
		grpcEndpoints:        grpcEndpoints,
		endpointsByURL:       make(map[string]*common.NodeUrl),
		optimizer:            optimizer,
		config:               config,
		rateLimiter:          NewGRPCClientRateLimiter(config),
		stickySessions:       make(map[string]string),
		clientSubscriptions:  make(map[string]map[string]struct{}),
		ctx:                  ctx,
		cancel:               cancel,
	}

	// Build endpoint lookup map
	for _, endpoint := range grpcEndpoints {
		manager.endpointsByURL[endpoint.Url] = endpoint
	}

	return manager
}

// Start initializes the manager and starts background tasks
func (dgm *DirectGRPCSubscriptionManager) Start(ctx context.Context) {
	utils.LavaFormatInfo("DirectGRPCSubscriptionManager starting",
		utils.LogAttr("chainID", dgm.chainID),
		utils.LogAttr("endpoints", len(dgm.grpcEndpoints)),
	)

	// Start cleanup goroutine
	go dgm.cleanupLoop(ctx)
}

// Stop gracefully shuts down the manager
func (dgm *DirectGRPCSubscriptionManager) Stop() {
	dgm.cancel()

	dgm.lock.Lock()
	defer dgm.lock.Unlock()

	// Close all active subscriptions
	for _, sub := range dgm.activeSubscriptions {
		sub.cancel()
		close(sub.closeSubChan)
	}
	dgm.activeSubscriptions = make(map[string]*grpcActiveSubscription)

	// Close all pools
	for _, pool := range dgm.upstreamPools {
		pool.Close()
	}
	dgm.upstreamPools = make(map[string]*UpstreamGRPCPool)

	utils.LavaFormatInfo("DirectGRPCSubscriptionManager stopped",
		utils.LogAttr("chainID", dgm.chainID),
	)
}

// cleanupLoop periodically cleans up stale subscriptions
func (dgm *DirectGRPCSubscriptionManager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(dgm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-dgm.ctx.Done():
			return
		case <-ticker.C:
			dgm.cleanupStaleSubscriptions()
		}
	}
}

// cleanupStaleSubscriptions removes subscriptions with cancelled contexts
func (dgm *DirectGRPCSubscriptionManager) cleanupStaleSubscriptions() {
	dgm.lock.Lock()
	defer dgm.lock.Unlock()

	var toRemove []string
	for hashedParams, sub := range dgm.activeSubscriptions {
		select {
		case <-sub.ctx.Done():
			toRemove = append(toRemove, hashedParams)
		default:
			// Still active
		}
	}

	for _, hashedParams := range toRemove {
		delete(dgm.activeSubscriptions, hashedParams)
		dgm.totalSubscriptions.Add(-1)
	}

	if len(toRemove) > 0 {
		utils.LavaFormatDebug("DirectGRPC: cleaned up stale subscriptions",
			utils.LogAttr("count", len(toRemove)),
		)
	}
}

// GetReflectionConnection returns a gRPC connection for reflection requests.
// This enables tools like grpcurl to discover services through the smart router.
// The cleanup function should be called when the connection is no longer needed.
func (dgm *DirectGRPCSubscriptionManager) GetReflectionConnection(ctx context.Context) (*grpc.ClientConn, func(), error) {
	if len(dgm.grpcEndpoints) == 0 {
		return nil, nil, fmt.Errorf("no gRPC endpoints available for reflection")
	}

	// Get a pool/connection for reflection
	pool, err := dgm.getOrCreatePool(ctx, dgm.grpcEndpoints[0])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pool for reflection: %w", err)
	}

	conn, err := pool.GetConnectionForStream(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get connection for reflection: %w", err)
	}

	// Return the underlying gRPC connection
	// The cleanup function doesn't need to do anything special since we're using the pool
	cleanup := func() {
		// Connection is managed by the pool - no cleanup needed
	}

	return conn.GetConn(), cleanup, nil
}

// IsStreamingMethod checks if a gRPC method is server-streaming
func (dgm *DirectGRPCSubscriptionManager) IsStreamingMethod(ctx context.Context, methodPath string) (bool, *desc.MethodDescriptor, error) {
	// Parse service and method name
	svc, methodName := rpcInterfaceMessages.ParseSymbol(methodPath)

	// Get a pool/connection to check the method descriptor
	pool, err := dgm.getOrCreatePool(ctx, dgm.grpcEndpoints[0])
	if err != nil {
		return false, nil, err
	}

	conn, err := pool.GetConnectionForStream(ctx)
	if err != nil {
		return false, nil, err
	}

	methodDesc, err := conn.GetMethodDescriptor(ctx, svc, methodName)
	if err != nil {
		return false, nil, err
	}

	return methodDesc.IsServerStreaming(), methodDesc, nil
}

// StartSubscription starts a new gRPC streaming subscription or joins an existing one
func (dgm *DirectGRPCSubscriptionManager) StartSubscription(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	dappID string,
	consumerIp string,
	connectionUniqueId string,
	metricsData *metrics.RelayMetrics,
) (*pairingtypes.RelayReply, <-chan *pairingtypes.RelayReply, error) {
	// Create client key for tracking
	clientKey := dgm.createClientKey(dappID, consumerIp, connectionUniqueId)

	// Rate limiting check
	if !dgm.rateLimiter.AllowSubscribe(clientKey) {
		return nil, nil, fmt.Errorf("subscription rate limit exceeded for client %s", clientKey)
	}

	// Get method path and request data
	grpcMessage, ok := chainMessage.GetRPCMessage().(*rpcInterfaceMessages.GrpcMessage)
	if !ok {
		return nil, nil, fmt.Errorf("expected GrpcMessage, got %T", chainMessage.GetRPCMessage())
	}

	methodPath := grpcMessage.Path
	if methodPath == "" {
		methodPath = chainMessage.GetApi().Name
	}
	requestData := grpcMessage.Msg

	// Create hash for deduplication
	hashedParams := dgm.hashSubscriptionParams(methodPath, requestData)

	// Check for existing subscription to join
	dgm.lock.RLock()
	existingSub, exists := dgm.activeSubscriptions[hashedParams]
	dgm.lock.RUnlock()

	if exists && dgm.config.SubscriptionSharingEnabled {
		return dgm.joinExistingSubscription(ctx, existingSub, clientKey, hashedParams)
	}

	// Check client subscription limit
	if err := dgm.checkClientSubscriptionLimit(clientKey); err != nil {
		return nil, nil, err
	}

	// Check global subscription limit
	if err := dgm.checkGlobalSubscriptionLimit(); err != nil {
		return nil, nil, err
	}

	// Create new subscription
	return dgm.createNewSubscription(ctx, chainMessage, methodPath, requestData, hashedParams, clientKey)
}

// joinExistingSubscription adds a client to an existing subscription
func (dgm *DirectGRPCSubscriptionManager) joinExistingSubscription(
	ctx context.Context,
	sub *grpcActiveSubscription,
	clientKey string,
	hashedParams string,
) (*pairingtypes.RelayReply, <-chan *pairingtypes.RelayReply, error) {
	sub.lock.Lock()
	defer sub.lock.Unlock()

	// Generate unique router ID for this client
	routerID := dgm.idMapper.GenerateRouterID(clientKey)
	dgm.idMapper.RegisterMapping(routerID, sub.routerSubscriptionID)

	// Create channel for this client
	replyChan := make(chan *pairingtypes.RelayReply, 100)
	sender := common.NewSafeChannelSender(ctx, replyChan)

	sub.clientRouterIDs[clientKey] = routerID
	sub.connectedClients[clientKey] = sender

	// Track subscription for this client
	dgm.trackClientSubscription(clientKey, hashedParams)

	utils.LavaFormatDebug("DirectGRPC: client joined existing subscription",
		utils.LogAttr("clientKey", clientKey),
		utils.LogAttr("routerID", routerID),
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	// Return first reply if available (for late joiners)
	firstReply := sub.firstReply
	if firstReply == nil {
		// Create acknowledgement
		firstReply = dgm.createStreamAcknowledgement(routerID, sub.methodPath)
	}

	return firstReply, replyChan, nil
}

// createNewSubscription creates a new upstream subscription
func (dgm *DirectGRPCSubscriptionManager) createNewSubscription(
	ctx context.Context,
	chainMessage chainlib.ChainMessage,
	methodPath string,
	requestData []byte,
	hashedParams string,
	clientKey string,
) (*pairingtypes.RelayReply, <-chan *pairingtypes.RelayReply, error) {
	// Select endpoint
	endpoint, err := dgm.selectEndpoint(clientKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to select endpoint: %w", err)
	}

	// Get or create pool
	pool, err := dgm.getOrCreatePool(ctx, endpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pool: %w", err)
	}

	// Get connection
	conn, err := pool.GetConnectionForStream(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Parse service and method
	svc, methodName := rpcInterfaceMessages.ParseSymbol(methodPath)

	// Get method descriptor
	methodDesc, err := conn.GetMethodDescriptor(ctx, svc, methodName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get method descriptor: %w", err)
	}

	// Verify it's a server-streaming method
	if !methodDesc.IsServerStreaming() {
		return nil, nil, fmt.Errorf("method %s is not a server-streaming method", methodPath)
	}

	// Create the upstream stream
	stream, err := dgm.createUpstreamStream(ctx, conn.GetConn(), methodPath, requestData, methodDesc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Increment stream count
	conn.IncrementStreams()

	// Create subscription context
	subCtx, subCancel := context.WithCancel(ctx)

	// Generate IDs
	routerSubID := dgm.idMapper.GenerateRouterID(clientKey)
	clientRouterID := dgm.idMapper.GenerateRouterID(clientKey)
	dgm.idMapper.RegisterMapping(clientRouterID, routerSubID)

	// Create reply channel for this client
	replyChan := make(chan *pairingtypes.RelayReply, 100)
	sender := common.NewSafeChannelSender(subCtx, replyChan)

	// Create active subscription
	activeSub := &grpcActiveSubscription{
		upstreamPool:         pool,
		upstreamConnection:   conn,
		upstreamStream:       stream,
		methodDescriptor:     methodDesc,
		routerSubscriptionID: routerSubID,
		hashedParams:         hashedParams,
		clientRouterIDs:      map[string]string{clientKey: clientRouterID},
		connectedClients:     map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]{clientKey: sender},
		methodPath:           methodPath,
		requestParams:        requestData,
		ctx:                  subCtx,
		cancel:               subCancel,
		closeSubChan:         make(chan struct{}),
	}

	// Store subscription
	dgm.lock.Lock()
	dgm.activeSubscriptions[hashedParams] = activeSub
	dgm.totalSubscriptions.Add(1)
	dgm.lock.Unlock()

	// Track for client
	dgm.trackClientSubscription(clientKey, hashedParams)

	// Start message listener
	go dgm.listenForUpstreamMessages(subCtx, hashedParams, activeSub)

	// Create acknowledgement as first reply
	firstReply := dgm.createStreamAcknowledgement(clientRouterID, methodPath)
	activeSub.firstReply = firstReply

	utils.LavaFormatInfo("DirectGRPC: created new subscription",
		utils.LogAttr("methodPath", methodPath),
		utils.LogAttr("clientKey", clientKey),
		utils.LogAttr("routerSubID", routerSubID),
		utils.LogAttr("endpoint", endpoint.Url),
	)

	return firstReply, replyChan, nil
}

// createUpstreamStream creates a gRPC client stream for server-streaming RPC
func (dgm *DirectGRPCSubscriptionManager) createUpstreamStream(
	ctx context.Context,
	conn *grpc.ClientConn,
	methodPath string,
	requestData []byte,
	methodDesc *desc.MethodDescriptor,
) (grpc.ClientStream, error) {
	// Create stream descriptor
	streamDesc := &grpc.StreamDesc{
		StreamName:    methodPath,
		ServerStreams: true,
		ClientStreams: false,
	}

	// Create the stream
	stream, err := conn.NewStream(ctx, streamDesc, "/"+methodPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	// Parse and send the initial request message
	if len(requestData) > 0 {
		msgFactory := dynamic.NewMessageFactoryWithDefaults()
		inputMsg := msgFactory.NewMessage(methodDesc.GetInputType())

		// Detect format and parse request data
		if len(requestData) > 0 && (requestData[0] == '{' || requestData[0] == '[') {
			// JSON input - use dynamic message's JSON unmarshaler
			if dynMsg, ok := inputMsg.(*dynamic.Message); ok {
				if err := dynMsg.UnmarshalJSON(requestData); err != nil {
					stream.CloseSend()
					return nil, fmt.Errorf("failed to parse JSON request: %w", err)
				}
			} else {
				stream.CloseSend()
				return nil, fmt.Errorf("unexpected message type for JSON parsing")
			}
		} else {
			// Binary proto input
			if err := proto.Unmarshal(requestData, inputMsg); err != nil {
				stream.CloseSend()
				return nil, fmt.Errorf("failed to parse proto request: %w", err)
			}
		}

		if err := stream.SendMsg(inputMsg); err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
	}

	// Close send direction (server-streaming is receive-only after initial request)
	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send: %w", err)
	}

	return stream, nil
}

// listenForUpstreamMessages listens for messages from upstream gRPC stream
func (dgm *DirectGRPCSubscriptionManager) listenForUpstreamMessages(
	ctx context.Context,
	hashedParams string,
	activeSub *grpcActiveSubscription,
) {
	defer func() {
		dgm.cleanupSubscription(hashedParams, activeSub)
	}()

	msgFactory := dynamic.NewMessageFactoryWithDefaults()

	for {
		select {
		case <-ctx.Done():
			return
		case <-activeSub.closeSubChan:
			return
		default:
			// Create fresh message for each receive
			outputMsg := msgFactory.NewMessage(activeSub.methodDescriptor.GetOutputType())

			// Receive next message
			err := activeSub.upstreamStream.RecvMsg(outputMsg)
			if err == io.EOF {
				utils.LavaFormatInfo("DirectGRPC: stream ended normally",
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				)
				return
			}
			if err != nil {
				utils.LavaFormatWarning("DirectGRPC: stream error",
					err,
					utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
				)
				// Attempt reconnection
				go dgm.handleUpstreamDisconnect(ctx, hashedParams, activeSub)
				return
			}

			// Marshal to bytes
			msgBytes, err := proto.Marshal(outputMsg)
			if err != nil {
				utils.LavaFormatWarning("DirectGRPC: failed to marshal message", err)
				continue
			}

			// Route to all clients
			dgm.routeMessageToClients(activeSub, msgBytes)
		}
	}
}

// routeMessageToClients routes upstream message to all connected clients
func (dgm *DirectGRPCSubscriptionManager) routeMessageToClients(
	activeSub *grpcActiveSubscription,
	msgData []byte,
) {
	activeSub.lock.RLock()
	clients := make(map[string]struct {
		sender   *common.SafeChannelSender[*pairingtypes.RelayReply]
		routerID string
	})
	for clientKey, sender := range activeSub.connectedClients {
		routerID := activeSub.clientRouterIDs[clientKey]
		clients[clientKey] = struct {
			sender   *common.SafeChannelSender[*pairingtypes.RelayReply]
			routerID string
		}{sender: sender, routerID: routerID}
	}
	activeSub.lock.RUnlock()

	seqNum := activeSub.messageSeq.Add(1)

	for _, info := range clients {
		reply := &pairingtypes.RelayReply{
			Data: msgData,
			Metadata: []pairingtypes.Metadata{
				{Name: MetadataGRPCSubscriptionID, Value: info.routerID},
				{Name: MetadataGRPCStreamSeq, Value: fmt.Sprintf("%d", seqNum)},
			},
		}
		info.sender.Send(reply)
	}
}

// handleUpstreamDisconnect handles upstream stream disconnection
func (dgm *DirectGRPCSubscriptionManager) handleUpstreamDisconnect(
	ctx context.Context,
	hashedParams string,
	activeSub *grpcActiveSubscription,
) {
	// Prevent concurrent restoration
	if !activeSub.restoring.CompareAndSwap(false, true) {
		return
	}
	defer activeSub.restoring.Store(false)

	utils.LavaFormatInfo("DirectGRPC: attempting to restore subscription",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	// Reconnect pool
	if err := activeSub.upstreamPool.ReconnectWithBackoff(ctx); err != nil {
		utils.LavaFormatWarning("DirectGRPC: failed to reconnect", err)
		activeSub.cancel()
		return
	}

	// Get new connection
	newConn, err := activeSub.upstreamPool.GetConnectionForStream(ctx)
	if err != nil {
		utils.LavaFormatWarning("DirectGRPC: failed to get new connection", err)
		activeSub.cancel()
		return
	}

	// Create new stream
	newStream, err := dgm.createUpstreamStream(
		ctx,
		newConn.GetConn(),
		activeSub.methodPath,
		activeSub.requestParams,
		activeSub.methodDescriptor,
	)
	if err != nil {
		utils.LavaFormatWarning("DirectGRPC: failed to create new stream", err)
		activeSub.cancel()
		return
	}

	// Update subscription
	activeSub.lock.Lock()
	oldConn := activeSub.upstreamConnection
	activeSub.upstreamConnection = newConn
	activeSub.upstreamStream = newStream
	activeSub.lock.Unlock()

	// Decrement old connection stream count
	if oldConn != nil {
		activeSub.upstreamPool.NotifyStreamRemoved(oldConn)
	}
	newConn.IncrementStreams()

	utils.LavaFormatInfo("DirectGRPC: subscription restored",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)

	// Restart listener
	go dgm.listenForUpstreamMessages(activeSub.ctx, hashedParams, activeSub)
}

// cleanupSubscription removes a subscription and notifies clients
func (dgm *DirectGRPCSubscriptionManager) cleanupSubscription(hashedParams string, activeSub *grpcActiveSubscription) {
	dgm.lock.Lock()
	delete(dgm.activeSubscriptions, hashedParams)
	dgm.totalSubscriptions.Add(-1)
	dgm.lock.Unlock()

	// Close client channels
	activeSub.lock.Lock()
	for _, sender := range activeSub.connectedClients {
		sender.Close()
	}
	activeSub.connectedClients = nil
	activeSub.lock.Unlock()

	// Return connection to pool
	if activeSub.upstreamConnection != nil {
		activeSub.upstreamPool.NotifyStreamRemoved(activeSub.upstreamConnection)
	}

	// Clean up ID mappings
	for _, routerID := range activeSub.clientRouterIDs {
		dgm.idMapper.RemoveMapping(routerID)
	}

	utils.LavaFormatDebug("DirectGRPC: subscription cleaned up",
		utils.LogAttr("hashedParams", utils.ToHexString(hashedParams)),
	)
}

// Unsubscribe handles unsubscribe request from a client
func (dgm *DirectGRPCSubscriptionManager) Unsubscribe(
	ctx context.Context,
	routerSubID string,
	clientKey string,
) error {
	// Rate limiting check
	if !dgm.rateLimiter.AllowUnsubscribe(clientKey) {
		return fmt.Errorf("unsubscribe rate limit exceeded")
	}

	// Find the subscription
	dgm.lock.RLock()
	var targetSub *grpcActiveSubscription
	var targetHashedParams string
	for hashedParams, sub := range dgm.activeSubscriptions {
		sub.lock.RLock()
		if _, exists := sub.clientRouterIDs[clientKey]; exists {
			if sub.clientRouterIDs[clientKey] == routerSubID {
				targetSub = sub
				targetHashedParams = hashedParams
			}
		}
		sub.lock.RUnlock()
		if targetSub != nil {
			break
		}
	}
	dgm.lock.RUnlock()

	if targetSub == nil {
		return fmt.Errorf("subscription not found for router ID %s", routerSubID)
	}

	return dgm.removeClientFromSubscription(targetSub, targetHashedParams, clientKey)
}

// UnsubscribeAll removes all subscriptions for a client
func (dgm *DirectGRPCSubscriptionManager) UnsubscribeAll(
	ctx context.Context,
	clientKey string,
) error {
	dgm.lock.RLock()
	clientSubs, exists := dgm.clientSubscriptions[clientKey]
	if !exists {
		dgm.lock.RUnlock()
		return nil
	}
	// Copy the set
	hashedParamsList := make([]string, 0, len(clientSubs))
	for hp := range clientSubs {
		hashedParamsList = append(hashedParamsList, hp)
	}
	dgm.lock.RUnlock()

	for _, hashedParams := range hashedParamsList {
		dgm.lock.RLock()
		sub, exists := dgm.activeSubscriptions[hashedParams]
		dgm.lock.RUnlock()
		if exists {
			dgm.removeClientFromSubscription(sub, hashedParams, clientKey)
		}
	}

	// Cleanup client tracking
	dgm.lock.Lock()
	delete(dgm.clientSubscriptions, clientKey)
	delete(dgm.stickySessions, clientKey)
	dgm.lock.Unlock()

	dgm.rateLimiter.CleanupClient(clientKey)

	return nil
}

// removeClientFromSubscription removes a client from a subscription
func (dgm *DirectGRPCSubscriptionManager) removeClientFromSubscription(
	sub *grpcActiveSubscription,
	hashedParams string,
	clientKey string,
) error {
	sub.lock.Lock()
	defer sub.lock.Unlock()

	// Remove client
	if sender, exists := sub.connectedClients[clientKey]; exists {
		sender.Close()
		delete(sub.connectedClients, clientKey)
	}

	routerID := sub.clientRouterIDs[clientKey]
	delete(sub.clientRouterIDs, clientKey)
	dgm.idMapper.RemoveMapping(routerID)

	// Untrack from client
	dgm.untrackClientSubscription(clientKey, hashedParams)

	// If last client, close the subscription
	if len(sub.connectedClients) == 0 {
		sub.cancel()
		close(sub.closeSubChan)
	}

	return nil
}

// Helper methods

func (dgm *DirectGRPCSubscriptionManager) createClientKey(dappID, consumerIp, connectionUniqueId string) string {
	return fmt.Sprintf("%s:%s:%s", dappID, consumerIp, connectionUniqueId)
}

func (dgm *DirectGRPCSubscriptionManager) hashSubscriptionParams(methodPath string, requestData []byte) string {
	return utils.ToHexString(fmt.Sprintf("%s:%s", methodPath, string(requestData)))
}

func (dgm *DirectGRPCSubscriptionManager) createStreamAcknowledgement(routerID, methodPath string) *pairingtypes.RelayReply {
	return &pairingtypes.RelayReply{
		Data: []byte(fmt.Sprintf(`{"subscription_id":"%s","method":"%s","status":"STREAMING"}`, routerID, methodPath)),
		Metadata: []pairingtypes.Metadata{
			{Name: MetadataGRPCSubscriptionID, Value: routerID},
			{Name: "content-type", Value: "application/json"},
		},
	}
}

func (dgm *DirectGRPCSubscriptionManager) getOrCreatePool(ctx context.Context, endpoint *common.NodeUrl) (*UpstreamGRPCPool, error) {
	dgm.lock.Lock()
	defer dgm.lock.Unlock()

	pool, exists := dgm.upstreamPools[endpoint.Url]
	if exists {
		return pool, nil
	}

	pool = NewUpstreamGRPCPoolWithConfig(endpoint, dgm.config)
	dgm.upstreamPools[endpoint.Url] = pool

	return pool, nil
}

func (dgm *DirectGRPCSubscriptionManager) selectEndpoint(clientKey string) (*common.NodeUrl, error) {
	// Check sticky session
	dgm.lock.RLock()
	stickyURL, hasSticky := dgm.stickySessions[clientKey]
	dgm.lock.RUnlock()

	if hasSticky {
		if endpoint, exists := dgm.endpointsByURL[stickyURL]; exists {
			return endpoint, nil
		}
	}

	// Select first available endpoint (could be enhanced with optimizer)
	if len(dgm.grpcEndpoints) == 0 {
		return nil, fmt.Errorf("no gRPC endpoints available")
	}

	endpoint := dgm.grpcEndpoints[0]

	// Set sticky session
	dgm.lock.Lock()
	dgm.stickySessions[clientKey] = endpoint.Url
	dgm.lock.Unlock()

	return endpoint, nil
}

func (dgm *DirectGRPCSubscriptionManager) checkClientSubscriptionLimit(clientKey string) error {
	dgm.lock.RLock()
	count := len(dgm.clientSubscriptions[clientKey])
	dgm.lock.RUnlock()

	if count >= dgm.config.MaxSubscriptionsPerClient {
		if dgm.config.ShouldRejectOnClientLimit() {
			return fmt.Errorf("client subscription limit reached (%d)", dgm.config.MaxSubscriptionsPerClient)
		}
		utils.LavaFormatWarning("DirectGRPC: client near subscription limit",
			nil,
			utils.LogAttr("clientKey", clientKey),
			utils.LogAttr("count", count),
		)
	}
	return nil
}

func (dgm *DirectGRPCSubscriptionManager) checkGlobalSubscriptionLimit() error {
	total := dgm.totalSubscriptions.Load()
	if total >= int64(dgm.config.MaxTotalSubscriptions) {
		if dgm.config.ShouldRejectOnTotalLimit() {
			return fmt.Errorf("global subscription limit reached (%d)", dgm.config.MaxTotalSubscriptions)
		}
		utils.LavaFormatWarning("DirectGRPC: approaching global subscription limit",
			nil,
			utils.LogAttr("total", total),
		)
	}
	return nil
}

func (dgm *DirectGRPCSubscriptionManager) trackClientSubscription(clientKey, hashedParams string) {
	dgm.lock.Lock()
	defer dgm.lock.Unlock()

	if dgm.clientSubscriptions[clientKey] == nil {
		dgm.clientSubscriptions[clientKey] = make(map[string]struct{})
	}
	dgm.clientSubscriptions[clientKey][hashedParams] = struct{}{}
}

func (dgm *DirectGRPCSubscriptionManager) untrackClientSubscription(clientKey, hashedParams string) {
	dgm.lock.Lock()
	defer dgm.lock.Unlock()

	if subs, exists := dgm.clientSubscriptions[clientKey]; exists {
		delete(subs, hashedParams)
		if len(subs) == 0 {
			delete(dgm.clientSubscriptions, clientKey)
		}
	}
}

// GetActiveSubscriptionCount returns the number of active subscriptions
func (dgm *DirectGRPCSubscriptionManager) GetActiveSubscriptionCount() int64 {
	return dgm.totalSubscriptions.Load()
}

// GetClientSubscriptionCount returns the number of subscriptions for a client
func (dgm *DirectGRPCSubscriptionManager) GetClientSubscriptionCount(clientKey string) int {
	dgm.lock.RLock()
	defer dgm.lock.RUnlock()
	return len(dgm.clientSubscriptions[clientKey])
}
