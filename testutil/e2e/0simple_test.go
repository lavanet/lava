package main

import (
	"testing"
)

func TestSimple(t *testing.T) {
	finalresults, err := SimpleTest(t)
	wrapGoTest(t, finalresults, err)
}

func simple_events() map[string](func(LogLine) TestResult) {
	tests := map[string](func(LogLine) TestResult){
		"NICE":  test_found_pass,
		"error": test_found_fail,
	}
	return tests
}

func nice_check(line string) TestResult {
	return test_basic(line, "NICE")
}

var simpleTest = TestProc{
	expectedEvents:   []string{"NICE"},
	unexpectedEvents: []string{"error", "x", "no"},
	tests:            simple_events(),
	strict:           true}

func SimpleTest(t *testing.T) ([]TestResult, error) {
	prepTest(t)

	simple := TestProcess("simple", "go run "+homepath+"testutil/e2e/simple/simple.go ", simpleTest)
	await(simple, "nice!", nice_check, "awaiting for NICE to proceed...")

	println("::::::::::::::::::::::::::::::::::::::::::::::")
	println("::::::::::::::::::::::::::::::::::::::::::::::")
	println("::::::::::::::::::::::::::::::::::::::::::::::")

	// Finalize & Display Results
	final := finalizeResults(t)

	return final, nil
}
