package statetracker

import (
	"context"
	"testing"

	"github.com/lavanet/lava/protocol/statetracker/mock"
	"github.com/stretchr/testify/assert"
)

func TestPaymentUpdater_UpdaterKey(t *testing.T) {
	// Initialize the PaymentUpdater object
	pu := NewPaymentUpdater(nil)

	// Ensure the UpdaterKey() function returns the correct key
	assert.Equal(t, pu.UpdaterKey(), CallbackKeyForPaymentUpdate)
}

func TestPaymentUpdater_RegisterEpochUpdatable(t *testing.T) {
	// Initialize the PaymentUpdater object
	pu := NewPaymentUpdater(nil)

	// Register new payment updater
	pu.RegisterPaymentUpdatable(context.Background(), mock.MockPaymentUpdatable{})

	// Ensure the new updater was added
	assert.Equal(t, len(pu.paymentUpdatables), 1)
}

func TestPaymentUpdater_Update(t *testing.T) {
	// Initialize the PaymentUpdater object
	pu := NewPaymentUpdater(mock.MockProviderStateQuery{})

	// Register new payment updater
	pu.RegisterPaymentUpdatable(context.Background(), mock.MockPaymentUpdatable{})

	// Update
	pu.Update(10)

	// Initialize the EpochUpdater object with error
	pu = NewPaymentUpdater(mock.MockErrorProviderStateQuery{})

	// Update
	pu.Update(10)
}
