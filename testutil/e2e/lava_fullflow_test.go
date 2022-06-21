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
	finalresults, err := XTestX(t)
	if err != nil {
		t.Errorf("expected no error but got %v", err)
	}
	for _, res := range finalresults {
		end := res.line
		if res.err != nil {
			if !res.passed {
				t.Errorf(res.err.Error())
			}
			end = res.err.Error() + " :  " + res.line
		}
		if res.passed {
			t.Log(" ::: PASSED ::: " + res.parent + " ::: " + res.eventID + " ::: " + end)
		} else {
			t.Log(" ::: FAILED ::: " + res.parent + " ::: " + res.eventID + " ::: " + end)
			t.Fail()
		}
	}
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

func XTestX(t *testing.T) ([]TestResult, error) {
	return FullFlowTest(t)
	// return InitTest(t)
}
