package main

import (
	"fmt"
	"testing"
)

func TestSimple(t *testing.T) {
	finalresults, err := SimpleTest(t)
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

func simple_tests() map[string](func(string) (bool, string, error)) {
	tests := map[string](func(string) (bool, string, error)){
		"NICE":  test_found_pass,
		"error": test_found_fail,
	}
	return tests
}

func nice(line string) (bool, bool) {
	contains := "NICE"
	return advanceFlow(line, contains), true
}

func SimpleTest(t *testing.T) ([]TestResult, error) {
	simpleTest := Test{
		expectedEvents:   []string{"NICE"},
		unexpectedEvents: []string{"error", "x", "no"},
		tests:            simple_tests(),
		strict:           true}

	testfailed := false
	failed := &testfailed
	states := []State{}
	results := map[string][]TestResult{}
	// homepath := getHomePath() + "/home/runner/work/goTestGH/goTestGH/"
	// homepath := "/home/runner/work/goTestGH/goTestGH/"
	homepath := getHomePath() + "go/lava/" //local
	// homepath := getHomePath() + "work/lava/lava/" //github
	fmt.Errorf("SSSSSSSSSSSSSS XXXXXXXXXXXXXXXXXXXXX HOME %s", getHomePath())

	node := LogProcess(CMD{
		stateID:  "simple",
		homepath: homepath,
		// cmd:          "go run " + homepath + "testutil/e2e/simple/simple.go ",
		cmd:          "go run ./testutil/e2e/simple/simple.go ",
		filter:       []string{"!", "no", "un", "error"},
		testing:      true,
		test:         simpleTest,
		results:      &results,
		dep:          nil,
		failed:       failed,
		requireAlive: true}, t, &states)
	await(node, "nice!", nice, "awating for NICE to proceed...")

	println("::::::::::::::::::::::::::::::::::::::::::::::")
	println("::::::::::::::::::::::::::::::::::::::::::::::")
	println("::::::::::::::::::::::::::::::::::::::::::::::")

	// Display Results

	fmt.Println(string("================================================="))
	fmt.Println(string("================ TEST DONE! ====================="))
	fmt.Println(string("================================================="))
	fmt.Println(string("=============== Expected Events ================="))

	finalExpectedTests := []TestResult{}
	finalUnexpectedTests := []TestResult{}
	count := 1
	for _, expected := range simpleTest.expectedEvents {
		// fmt.Println("XXX " + expected)
		if testList, foundEvent := results[expected]; foundEvent {
			// fmt.Println("foundxxxxx")
			for _, res := range testList {
				finalExpectedTests = append(finalExpectedTests, res)
				printTestResult(res, t)
				// fmt.Println("RRRRRRRRRRR")
				count += 1
			}
		}
		delete(results, expected)
	}

	count = 1
	fmt.Println(string("============== Unexpected Events ================"))
	for key, testList := range results {
		for _, res := range testList {
			count += 1
			printTestResult(res, t)
			finalUnexpectedTests = append(finalUnexpectedTests, res)
		}
		delete(results, key)
	}
	fmt.Println(string("================================================="))
	if *failed && t != nil {
		t.Errorf(" ::: Test Failed ::: ")
		t.FailNow()
	}
	// nothing([]State{init, client}) // we are not using client or init atm so theres a "declared but not used" error
	// exit(states)
	final := append(finalExpectedTests, finalUnexpectedTests...)
	return final, nil
}
