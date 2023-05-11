package statetracker

import (
	"sync"

	"github.com/lavanet/lava/protocol/rpcprovider/rewardserver"
	"golang.org/x/net/context"
)

const (
	CallbackKeyForPaymentUpdate = "payment-update"
)

type PaymentUpdatable interface {
	PaymentHandler(*rewardserver.PaymentRequest)
	Description() string
}

type PaymentUpdater struct {
	lock             sync.RWMutex
	paymentUpdatable map[string]*PaymentUpdatable
	stateQuery       *ProviderStateQuery
}

func NewPaymentUpdater(stateQuery *ProviderStateQuery) *PaymentUpdater {
	return &PaymentUpdater{paymentUpdatable: map[string]*PaymentUpdatable{}, stateQuery: stateQuery}
}

func (pu *PaymentUpdater) RegisterPaymentUpdatable(ctx context.Context, paymentUpdatable *PaymentUpdatable) {
	pu.lock.Lock()
	defer pu.lock.Unlock()
	pu.paymentUpdatable[(*paymentUpdatable).Description()] = paymentUpdatable
}

func (pu *PaymentUpdater) UpdaterKey() string {
	return CallbackKeyForPaymentUpdate
}

func (pu *PaymentUpdater) Update(latestBlock int64) {
	pu.lock.RLock()
	defer pu.lock.RUnlock()
	ctx := context.Background()
	payments, err := pu.stateQuery.PaymentEvents(ctx, latestBlock)
	if err != nil {
		return
	}
	for _, payment := range payments {
		updatable, foundUpdatable := pu.paymentUpdatable[payment.Description]
		if foundUpdatable {
			(*updatable).PaymentHandler(payment)
		}
	}
}
