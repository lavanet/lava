package statetracker

import (
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
	paymentUpdatable map[string]*PaymentUpdatable
	stateQuery       *ProviderStateQuery
	eventTracker     *EventTracker
}

func NewPaymentUpdater(stateQuery *ProviderStateQuery, eventTracker *EventTracker) *PaymentUpdater {
	return &PaymentUpdater{paymentUpdatable: map[string]*PaymentUpdatable{}, stateQuery: stateQuery, eventTracker: eventTracker}
}

func (pu *PaymentUpdater) RegisterPaymentUpdatable(ctx context.Context, paymentUpdatable *PaymentUpdatable) {
	pu.paymentUpdatable[(*paymentUpdatable).Description()] = paymentUpdatable
}

func (pu *PaymentUpdater) UpdaterKey() string {
	return CallbackKeyForPaymentUpdate
}

func (pu *PaymentUpdater) Update(latestBlock int64) {
	payments, err := pu.eventTracker.getLatestPaymentEvents()
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
