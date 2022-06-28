package main

import (
	"fmt"
	"os"
	"strings"
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

func simple_tests() map[string](func(LogLine) TestResult) {
	tests := map[string](func(LogLine) TestResult){
		"NICE":  test_found_pass,
		"error": test_found_fail,
	}
	return tests
}

func nice(line string) TestResult {
	return test_basic(line, "NICE")
}

func SimpleTest(t *testing.T) ([]TestResult, error) {
	simpleTest := TestProcess{
		expectedEvents:   []string{"NICE"},
		unexpectedEvents: []string{"error", "x", "no"},
		tests:            simple_tests(),
		strict:           true}

	testfailed := false
	failed := &testfailed
	states := []State{}
	results := map[string][]TestResult{}
	homepath := os.Getenv("LAVA")
	if homepath == "" {
		homepath = getHomePath()
		if strings.Contains(homepath, "runner") { // on github
			homepath += "work/lava/lava"
		} else {
			homepath += "go/lava" //local
		}
	}
	homepath += "/"

	if t != nil {
		t.Logf(" ::: Test Homepath ::: %s", homepath)
	}
	node := LogProcess(CMD{
		stateID:      "simple",
		homepath:     homepath,
		cmd:          "go run " + homepath + "testutil/e2e/simple/simple.go ",
		filter:       []string{"!", "no", "un", "error"},
		testing:      true,
		test:         simpleTest,
		results:      &results,
		dep:          nil,
		failed:       failed,
		requireAlive: true}, t, &states)
	await(node, "nice!", nice, "awaiting for NICE to proceed...")

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
		if testList, foundEvent := results[expected]; foundEvent {
			for _, res := range testList {
				finalExpectedTests = append(finalExpectedTests, res)
				printTestResult(res, t)
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
	final := append(finalExpectedTests, finalUnexpectedTests...)
	return final, nil
}
