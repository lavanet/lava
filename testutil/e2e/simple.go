package main

import (
	"fmt"
	"testing"
)

func nice(line string) (bool, bool) {
	contains := "NICE"
	return advanceFlow(line, contains), true
}

func SimpleTest(t *testing.T) ([]TestResult, error) {
	simpleTest := Test{
		expectedEvents:   []string{"NICE"},
		unexpectedEvents: []string{"error", "x"},
		tests:            tests(),
		strict:           true}

	testfailed := false
	failed := &testfailed
	states := []State{}
	results := map[string][]TestResult{}
	// homepath := getHomePath() + "/home/runner/work/goTestGH/goTestGH/"
	// homepath := "/home/runner/work/goTestGH/goTestGH/"
	homepath := getHomePath() + "work/lava/lava/" //github
	fmt.Errorf("XXXXXXXXXXXXXXXXXXXXX HOME %s", getHomePath())

	node := LogProcess(CMD{
		stateID:      "simple",
		homepath:     homepath,
		cmd:          "go run " + homepath + "simple.go ",
		filter:       []string{"!"},
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
