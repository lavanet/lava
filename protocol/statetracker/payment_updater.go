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
	paymentUpdatables map[string]*PaymentUpdatable
	stateQuery        *ProviderStateQuery
}

func NewPaymentUpdater(stateQuery *ProviderStateQuery) *PaymentUpdater {
	return &PaymentUpdater{paymentUpdatables: map[string]*PaymentUpdatable{}, stateQuery: stateQuery}
}

func (pu *PaymentUpdater) RegisterPaymentUpdatable(ctx context.Context, paymentUpdatable *PaymentUpdatable) {
	pu.paymentUpdatables[(*paymentUpdatable).Description()] = paymentUpdatable
}

func (pu *PaymentUpdater) UpdaterKey() string {
	return CallbackKeyForPaymentUpdate
}

func (pu *PaymentUpdater) Update(latestBlock int64) {
	ctx := context.Background()
	payments, err := pu.stateQuery.PaymentEvents(ctx, latestBlock)
	if err != nil {
		return
	}
	for _, payment := range payments {
		updatable := pu.paymentUpdatables[payment.Description]
		(*updatable).PaymentHandler(payment)
	}
}
