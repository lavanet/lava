package main

import (
	"testing"
)

func TestFullflow(t *testing.T) {
	finalresults, err := FullFlowTest(t)
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
