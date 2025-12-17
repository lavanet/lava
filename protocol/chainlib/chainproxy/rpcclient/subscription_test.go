// protocol/chainlib/chainproxy/rpcclient/subscription_test.go

package rpcclient

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestClientSubscription_UnsubscribeBeforeRun tests that calling Unsubscribe()
// before the subscription's run() goroutine starts doesn't deadlock.
// This validates the buffered quit channel fix from PR #2139.
func TestClientSubscription_UnsubscribeBeforeRun(t *testing.T) {
	// Create a minimal mock client
	client := &Client{
		idgen: func() ID { return ID("test-id") },
	}

	// Create a channel for subscription results
	resultCh := make(chan interface{}, 1)
	resultValue := reflect.ValueOf(resultCh)

	// Create the subscription but DON'T start sub.run()
	sub := newClientSubscription(client, "test", resultValue)

	// Verify quit channel is buffered
	require.Equal(t, 1, cap(sub.quit), "quit channel should be buffered with capacity 1")

	// Call Unsubscribe immediately without starting the forwarding goroutine
	// This should NOT deadlock thanks to the buffered quit channel
	done := make(chan struct{})
	go func() {
		sub.Unsubscribe()
		close(done)
	}()

	// Wait with timeout to detect deadlock
	select {
	case <-done:
		// Success - Unsubscribe completed without blocking
		t.Log("✓ Unsubscribe completed without deadlock")
	case <-time.After(1 * time.Second):
		t.Fatal("Unsubscribe deadlocked when called before run() started")
	}

	// Verify error channel is closed
	select {
	case _, ok := <-sub.Err():
		require.False(t, ok, "error channel should be closed after Unsubscribe")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("error channel not closed after Unsubscribe")
	}
}

// TestClientSubscription_UnsubscribeRace tests concurrent Unsubscribe calls
// to ensure they're idempotent and don't panic or deadlock.
func TestClientSubscription_UnsubscribeRace(t *testing.T) {
	client := &Client{
		idgen: func() ID { return ID("test-id") },
	}

	resultCh := make(chan interface{}, 1)
	sub := newClientSubscription(client, "test", reflect.ValueOf(resultCh))

	// Call Unsubscribe from multiple goroutines simultaneously
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			sub.Unsubscribe()
		}()
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("✓ Concurrent Unsubscribe calls completed safely")
	case <-time.After(2 * time.Second):
		t.Fatal("Concurrent Unsubscribe calls deadlocked")
	}
}

// TestClientSubscription_UnsubscribeWithRunning tests normal operation where
// Unsubscribe is called while the subscription is actively running.
func TestClientSubscription_UnsubscribeWithRunning(t *testing.T) {
	client := &Client{
		idgen: func() ID { return ID("test-id") },
	}

	resultCh := make(chan string, 10)
	sub := newClientSubscription(client, "test", reflect.ValueOf(resultCh))

	// Start the subscription forwarding goroutine
	var runComplete sync.WaitGroup
	runComplete.Add(1)
	
	go func() {
		defer runComplete.Done()
		sub.run()
	}()

	// Give the goroutine time to start
	time.Sleep(50 * time.Millisecond)

	// Send a quit signal via Unsubscribe
	sub.Unsubscribe()

	// Verify that run() completes
	done := make(chan struct{})
	go func() {
		runComplete.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("✓ Subscription run() completed after Unsubscribe")
	case <-time.After(2 * time.Second):
		t.Fatal("Subscription run() didn't exit after Unsubscribe")
	}

	// Verify error channel is closed
	_, ok := <-sub.Err()
	require.False(t, ok, "error channel should be closed after Unsubscribe")
}

// TestClientSubscription_BufferedQuitChannelPreventsDeadlock specifically tests
// the scenario where quit channel being buffered prevents the deadlock described
// in the PR #2139 fix comments.
func TestClientSubscription_BufferedQuitChannelPreventsDeadlock(t *testing.T) {
	client := &Client{
		idgen: func() ID { return ID("test-id") },
	}

	resultCh := make(chan interface{}, 1)
	sub := newClientSubscription(client, "test", reflect.ValueOf(resultCh))

	// Scenario: Unsubscribe called during connection setup, before run() starts
	// With unbuffered channel: Unsubscribe blocks on `quit <- err`, deadlock
	// With buffered channel: Message queued, Unsubscribe returns immediately

	unsubscribeStarted := make(chan struct{})
	unsubscribeComplete := make(chan struct{})

	go func() {
		close(unsubscribeStarted)
		sub.Unsubscribe()
		close(unsubscribeComplete)
	}()

	// Wait for Unsubscribe to be called
	<-unsubscribeStarted

	// Verify Unsubscribe completes quickly WITHOUT run() being started
	// This is the key test: with buffered channel, this should succeed
	select {
	case <-unsubscribeComplete:
		t.Log("✓ Buffered quit channel prevented deadlock")
		
		// Now verify the quit signal is still available if run() starts later
		select {
		case err := <-sub.quit:
			require.Equal(t, errUnsubscribed, err, "quit channel should contain unsubscribe error")
		default:
			t.Fatal("quit channel should have buffered the unsubscribe signal")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Buffered quit channel didn't prevent deadlock - Unsubscribe blocked")
	}
}

// TestClientSubscription_ErrChannelBehavior tests the error channel lifecycle
func TestClientSubscription_ErrChannelBehavior(t *testing.T) {
	client := &Client{
		idgen: func() ID { return ID("test-id") },
	}

	resultCh := make(chan interface{}, 1)
	sub := newClientSubscription(client, "test", reflect.ValueOf(resultCh))

	// Initially, error channel should block (no errors, not closed)
	select {
	case <-sub.Err():
		t.Fatal("error channel should not have any values initially")
	case <-time.After(10 * time.Millisecond):
		// Expected behavior
	}

	// After Unsubscribe, error channel should close
	sub.Unsubscribe()

	// Should be able to read immediately (closed channel)
	_, ok := <-sub.Err()
	require.False(t, ok, "error channel should be closed after Unsubscribe")

	// Multiple reads should succeed (closed channel property)
	_, ok = <-sub.Err()
	require.False(t, ok, "can read from closed error channel multiple times")
}
