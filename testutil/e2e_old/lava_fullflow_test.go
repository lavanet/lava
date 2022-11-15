package main

import (
	"testing"
	"time"
)

func TestLava(t *testing.T) {
	withTimeout := true
	if withTimeout {
		LavaTestFullflowWithTimeOut(t)
	} else {
		LavaTestFullflow(t)
	}
}

func LavaTestFullflow(t *testing.T) {
	finalresults, err := FullFlowTest(t)
	wrapGoTest(t, finalresults, err)
}

func LavaTestFullflowWithTimeOut(t *testing.T) {
	timeout := time.After(11 * time.Minute)
	done := make(chan bool)
	go func() {
		// run lava testing
		LavaTestFullflow(t)
		// Test finished on time !
		done <- true
	}()

	select {
	case <-timeout:
		t.Fatal("Test didn't finish in time")
	case <-done:
	}
}
