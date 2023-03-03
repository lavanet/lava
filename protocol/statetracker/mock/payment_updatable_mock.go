package mock

import "github.com/lavanet/lava/protocol/rpcprovider/rewardserver"

type MockPaymentUpdatable struct{}

func (m MockPaymentUpdatable) Description() string {
	return "paymentDescription"
}

func (m MockPaymentUpdatable) PaymentHandler(pr *rewardserver.PaymentRequest) {
	// Provide an implementation for PaymentHandler method
	// ...
}
