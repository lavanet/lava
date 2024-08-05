package updaters

import (
	"sync"

	"github.com/lavanet/lava/v2/protocol/rpcprovider/rewardserver"
	"github.com/lavanet/lava/v2/utils"
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
	eventTracker     *EventTracker
}

func NewPaymentUpdater(eventTracker *EventTracker) *PaymentUpdater {
	return &PaymentUpdater{paymentUpdatable: map[string]*PaymentUpdatable{}, eventTracker: eventTracker}
}

func (pu *PaymentUpdater) RegisterPaymentUpdatable(ctx context.Context, paymentUpdatable *PaymentUpdatable) {
	pu.lock.Lock()
	defer pu.lock.Unlock()
	pu.paymentUpdatable[(*paymentUpdatable).Description()] = paymentUpdatable
}

func (pu *PaymentUpdater) UpdaterKey() string {
	return CallbackKeyForPaymentUpdate
}

func (pu *PaymentUpdater) updateInner() {
	pu.lock.RLock()
	defer pu.lock.RUnlock()
	payments, err := pu.eventTracker.getLatestPaymentEvents()
	if err != nil {
		return
	}
	relevantPayments := 0
	for _, payment := range payments {
		updatable, foundUpdatable := pu.paymentUpdatable[payment.Description]
		if foundUpdatable {
			relevantPayments += 1
			(*updatable).PaymentHandler(payment)
		}
	}
	if len(payments) > 0 {
		utils.LavaFormatDebug("relevant payment events", utils.Attribute{Key: "number_of_payment_events_detected", Value: len(payments)}, utils.Attribute{Key: "number_of_relevant_payments_detected", Value: relevantPayments})
	}
}

func (pu *PaymentUpdater) Reset(latestBlock int64) {
	// in case we need a reset we don't have much to do as we might have lost some data due to pruning. we can just continue parsing our transactions
	// TODO: we can fetch the latest transactions for our account and see if we have some missing data but for that we need to keep track all transactions per provider account
	pu.updateInner()
}

func (pu *PaymentUpdater) Update(latestBlock int64) {
	pu.updateInner()
}
