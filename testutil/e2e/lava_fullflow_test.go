package main

import (
	"testing"
	"time"
)

func TestFullflow(t *testing.T) {
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
			t.Log(" ::: PASSED ::: " + res.eventID + " ::: " + end)
		} else {
			t.Log(" ::: FAILED ::: " + res.eventID + " ::: " + end)
			t.Fail()
		}
	}
}

func xTestFullflowWithTimeOut(t *testing.T) {
	// timeout := time.After(3 * time.Second)
	timeout := time.After(3 * time.Minute)
	done := make(chan bool)
	go func() {
		// do your testing
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
		// Test finished on time !
		// time.Sleep(5 * time.Second)
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
