package lavasession

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
)

type ProviderSessionsEpochData struct {
	UsedComputeUnits    uint64
	MaxComputeUnits     uint64
	MissingComputeUnits uint64
}

type NetworkAddressData struct {
	Address    string `yaml:"address,omitempty" json:"address,omitempty" mapstructure:"address,omitempty"` // HOST:PORT
	KeyPem     string `yaml:"key-pem,omitempty" json:"key-pem,omitempty" mapstructure:"key-pem"`
	CertPem    string `yaml:"cert-pem,omitempty" json:"cert-pem,omitempty" mapstructure:"cert-pem"`
	DisableTLS bool   `yaml:"disable-tls,omitempty" json:"disable-tls,omitempty" mapstructure:"disable-tls"`
}

type RPCProviderEndpoint struct {
	NetworkAddress NetworkAddressData `yaml:"network-address,omitempty" json:"network-address,omitempty" mapstructure:"network-address,omitempty"`
	ChainID        string             `yaml:"chain-id,omitempty" json:"chain-id,omitempty" mapstructure:"chain-id"` // spec chain identifier
	ApiInterface   string             `yaml:"api-interface,omitempty" json:"api-interface,omitempty" mapstructure:"api-interface"`
	Geolocation    uint64             `yaml:"geolocation,omitempty" json:"geolocation,omitempty" mapstructure:"geolocation"`
	NodeUrls       []common.NodeUrl   `yaml:"node-urls,omitempty" json:"node-urls,omitempty" mapstructure:"node-urls"`
}

func (endpoint *RPCProviderEndpoint) UrlsString() string {
	st_urls := make([]string, len(endpoint.NodeUrls))
	for idx, url := range endpoint.NodeUrls {
		st_urls[idx] = url.Url
	}
	return strings.Join(st_urls, ", ")
}

func (endpoint *RPCProviderEndpoint) AddonsString() string {
	st_urls := make([]string, len(endpoint.NodeUrls))
	for idx, url := range endpoint.NodeUrls {
		st_urls[idx] = strings.Join(url.Addons, ",")
	}
	return strings.Join(st_urls, "; ")
}

func (endpoint *RPCProviderEndpoint) String() string {
	return endpoint.ChainID + ":" + endpoint.ApiInterface + " Network Address:" + endpoint.NetworkAddress.Address + " Node: " + endpoint.UrlsString() + " Geolocation:" + strconv.FormatUint(endpoint.Geolocation, 10) + " Addons:" + endpoint.AddonsString()
}

func (endpoint *RPCProviderEndpoint) Validate() error {
	if len(endpoint.NodeUrls) == 0 {
		return utils.LavaFormatError("Empty URL list for endpoint", nil, utils.Attribute{Key: "endpoint", Value: endpoint.String()})
	}
	for _, url := range endpoint.NodeUrls {
		err := common.ValidateEndpoint(url.Url, endpoint.ApiInterface)
		if err != nil {
			return err
		}
	}
	return nil
}

type dataHandler interface {
	onDeleteEvent()
}

type subscriptionData struct {
	subscriptionMap map[string]map[string]*RPCSubscription
}

func (sm subscriptionData) onDeleteEvent() {
	for _, consumer := range sm.subscriptionMap {
		for _, subscription := range consumer {
			if subscription.Sub == nil { // validate subscription not nil
				utils.LavaFormatError("filterOldEpochEntriesSubscribe Error", SubscriptionPointerIsNilError, utils.Attribute{Key: "subscripionId", Value: subscription.Id})
			} else {
				subscription.Sub.Unsubscribe()
			}
		}
	}
}

type sessionData struct {
	sessionMap map[string]*ProviderSessionsWithConsumer
}

func (sm sessionData) onDeleteEvent() { // do nothing
}

type RPCSubscription struct {
	Id                   string
	Sub                  *rpcclient.ClientSubscription
	SubscribeRepliesChan chan interface{}
}

func (rpcpe *RPCProviderEndpoint) Key() string {
	return rpcpe.ChainID + rpcpe.ApiInterface
}

const (
	notBlockListedConsumer = 0
	blockListedConsumer    = 1
	notDataReliabilityPSWC = 0
	isDataReliabilityPSWC  = 1
)

// holds all of the data for a consumer for a certain epoch
type ProviderSessionsWithConsumer struct {
	Sessions          map[uint64]*SingleProviderSession
	isBlockListed     uint32
	consumerAddr      string
	epochData         *ProviderSessionsEpochData
	badgeEpochData    map[string]*ProviderSessionsEpochData
	Lock              sync.RWMutex
	isDataReliability uint32 // 0 is false, 1 is true. set to uint so we can atomically read
	pairedProviders   int64
}

type BadgeSession struct {
	BadgeCuAllocation uint64
	BadgeUser         string
}

func NewProviderSessionsWithConsumer(consumerAddr string, epochData *ProviderSessionsEpochData, isDataReliability uint32, pairedProviders int64) *ProviderSessionsWithConsumer {
	pswc := &ProviderSessionsWithConsumer{
		Sessions:          map[uint64]*SingleProviderSession{},
		isBlockListed:     0,
		consumerAddr:      consumerAddr,
		epochData:         epochData,
		badgeEpochData:    map[string]*ProviderSessionsEpochData{},
		isDataReliability: isDataReliability,
		pairedProviders:   pairedProviders,
	}
	return pswc
}

// reads the pairedProviders data atomically for DR
func (pswc *ProviderSessionsWithConsumer) atomicReadPairedProviders() int64 {
	return atomic.LoadInt64(&pswc.pairedProviders)
}

// reads the isDataReliability data atomically
func (pswc *ProviderSessionsWithConsumer) atomicReadIsDataReliability() uint32 {
	return atomic.LoadUint32(&pswc.isDataReliability)
}

// reads cs.BlockedEpoch atomically to determine if the consumer is blocked notBlockListedConsumer = 0, blockListedConsumer = 1
func (pswc *ProviderSessionsWithConsumer) atomicReadConsumerBlocked() (blockStatus uint32) {
	return atomic.LoadUint32(&pswc.isBlockListed)
}

func (pswc *ProviderSessionsWithConsumer) atomicReadMaxComputeUnits() (maxComputeUnits uint64) {
	return atomic.LoadUint64(&pswc.epochData.MaxComputeUnits)
}

func atomicReadBadgeMaxComputeUnits(badgeUserEpochData *ProviderSessionsEpochData) (maxComputeUnits uint64) {
	return atomic.LoadUint64(&badgeUserEpochData.MaxComputeUnits)
}

func (pswc *ProviderSessionsWithConsumer) atomicReadUsedComputeUnits() (usedComputeUnits uint64) {
	return atomic.LoadUint64(&pswc.epochData.UsedComputeUnits)
}

func (pswc *ProviderSessionsWithConsumer) atomicWriteUsedComputeUnits(cu uint64) {
	atomic.StoreUint64(&pswc.epochData.UsedComputeUnits, cu)
}

func atomicReadBadgeUsedComputeUnits(badgeUserEpochData *ProviderSessionsEpochData) (usedComputeUnits uint64) {
	return atomic.LoadUint64(&badgeUserEpochData.UsedComputeUnits)
}

func (pswc *ProviderSessionsWithConsumer) atomicCompareAndWriteUsedComputeUnits(newUsed uint64, knownUsed uint64) bool {
	if newUsed == knownUsed { // no need to compare swap
		return true
	}
	return atomic.CompareAndSwapUint64(&pswc.epochData.UsedComputeUnits, knownUsed, newUsed)
}

func atomicCompareAndWriteBadgeUsedComputeUnits(newUsed uint64, knownUsed uint64, badgeUserEpochData *ProviderSessionsEpochData) bool {
	if newUsed == knownUsed { // no need to compare swap
		return true
	}
	return atomic.CompareAndSwapUint64(&badgeUserEpochData.UsedComputeUnits, knownUsed, newUsed)
}

func (pswc *ProviderSessionsWithConsumer) atomicReadMissingComputeUnits() (missingComputeUnits uint64) {
	return atomic.LoadUint64(&pswc.epochData.MissingComputeUnits)
}

func (pswc *ProviderSessionsWithConsumer) atomicCompareAndWriteMissingComputeUnits(newUsed uint64, knownUsed uint64) bool {
	if newUsed == knownUsed { // no need to compare swap
		return true
	}
	return atomic.CompareAndSwapUint64(&pswc.epochData.MissingComputeUnits, knownUsed, newUsed)
}

func (pswc *ProviderSessionsWithConsumer) SafeAddMissingComputeUnits(currentMissingCU uint64, allowedThreshold float64) (legitimate bool, totalMissingCu uint64) {
	for {
		missing := pswc.atomicReadMissingComputeUnits()
		used := pswc.atomicReadUsedComputeUnits()
		max := pswc.atomicReadMaxComputeUnits()
		totalMissingCu = missing + currentMissingCU
		// do not allow bypassing max used CU
		if totalMissingCu+used > max {
			return false, totalMissingCu
		}
		// do not allow having more missing than threshold
		if totalMissingCu > uint64(float64(max)*allowedThreshold) {
			return false, totalMissingCu
		}
		// do not allow having more missing than already used
		if totalMissingCu > used {
			return false, totalMissingCu
		}
		if pswc.atomicCompareAndWriteMissingComputeUnits(totalMissingCu, missing) {
			return true, totalMissingCu
		}
	}
}

// create a new session with a consumer, and store it inside it's providerSessions parent
func (pswc *ProviderSessionsWithConsumer) createNewSingleProviderSession(ctx context.Context, sessionId uint64, epoch uint64) (session *SingleProviderSession, err error) {
	utils.LavaFormatDebug("Provider creating new sessionID", utils.Attribute{Key: "SessionID", Value: sessionId}, utils.Attribute{Key: "epoch", Value: epoch})
	session = &SingleProviderSession{
		userSessionsParent: pswc,
		SessionID:          sessionId,
		PairingEpoch:       epoch,
	}
	pswc.Lock.Lock()
	defer pswc.Lock.Unlock()

	// this is a double lock and risky but we just created session and nobody has reference to it yet
	// the following code has to be as short as possible
	session.lockForUse(ctx)
	pswc.Sessions[sessionId] = session
	// session is still locked when we return it
	return session, nil
}

// this function returns the session locked to be used
func (pswc *ProviderSessionsWithConsumer) getExistingSession(ctx context.Context, sessionId uint64) (session *SingleProviderSession, err error) {
	pswc.Lock.RLock()
	defer pswc.Lock.RUnlock()
	if session, ok := pswc.Sessions[sessionId]; ok {
		err := session.tryLockForUse(ctx)
		return session, err
	}
	return nil, SessionDoesNotExist
}

// this function verifies the provider can create a data reliability session and returns one if valid
func (pswc *ProviderSessionsWithConsumer) getDataReliabilitySingleSession(sessionId uint64, epoch uint64) (session *SingleProviderSession, err error) {
	utils.LavaFormatDebug("Provider creating new DataReliabilitySingleSession", utils.Attribute{Key: "SessionID", Value: sessionId}, utils.Attribute{Key: "epoch", Value: epoch})
	session, foundDataReliabilitySession := pswc.Sessions[sessionId]
	if foundDataReliabilitySession {
		// if session exists, relay number should be 0 as it might have had an error
		// locking the session and returning for validation
		session.lock.Lock()
		return session, nil
	}

	// otherwise return a new session and add it to the sessions list
	session = &SingleProviderSession{
		userSessionsParent: pswc,
		SessionID:          sessionId,
		PairingEpoch:       epoch,
	}
	pswc.Lock.Lock()
	defer pswc.Lock.Unlock()
	// this is a double lock and risky but we just created session and nobody has reference to it yet
	session.lock.Lock()
	pswc.Sessions[sessionId] = session

	// session is still locked when we return it
	return session, nil
}
