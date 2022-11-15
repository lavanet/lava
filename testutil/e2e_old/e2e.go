package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

var results = map[string][]*TestResult{}
var homepath, isGithubAction = prepHomePath()
var testfailed = false
var failed = &testfailed
var states = []State{}
var options = map[string]*bool{}
var tt *testing.T = nil

func TestProcess(id string, exec string, test TestProc) (state State) {
	cmd := CMD{}
	cmd.stateID = id
	cmd.cmd = exec
	cmd.testing = true
	cmd.dep = nil
	cmd.requireAlive = false
	cmd.results = results
	cmd.homepath = homepath
	cmd.failed = failed
	cmd.test = test
	cmd.filter = test.filter
	debug := false
	stdout := true
	options[id+"_debug"] = &debug
	options[id+"_stdout"] = &stdout
	cmd.debug = options[id+"_debug"]
	cmd.stdout = options[id+"_stdout"]
	// cmd = options[id+"_debug"]

	state = LogProcess(cmd, tt, &states)
	return
}

func awaitState(state State) {
	fin := false
	markForDelete := []string{}
	if !*state.failed {
		for !fin {
			if *state.finished || *state.failed {
				break
			}
			if len(state.awaiting) == 0 {
				fin = true
				break
			} else {
				time.Sleep(1 * time.Second)
				for key, await := range state.awaiting {
					if !*state.failed {
						last := " :::(log tail)::: " + string(*state.lastLine)
						println(" ::: " + state.id + " ::: " + await.msg + last)
					}
					if *await.done {
						markForDelete = append(markForDelete, key)
					}
					if *state.finished {
						break
					}
				}
				for _, key := range markForDelete {
					if _, exist := state.awaiting[key]; exist {
						delete(state.awaiting, key)
					}
				}
			}
		}
	}
	if !*state.finished && !*state.failed {
		println("*********************************************")
		for _, key := range markForDelete {
			println(" ::: DONE ::: " + state.id + " ::: " + key)
		}
		println("*********************************************")
	}

}

func await(state State, id string, f func(string) TestResult, msg string) {
	done, pass := false, false
	state.awaiting[id] = Await{&done, &pass, f, msg}
	awaitState(state)
}

func getExpectedEvents(states []State) []string {
	expected := []string{}
	for _, state := range states {
		expected = append(expected, state.test.expectedEvents...)
	}
	return expected
}

func readFile(path string, state State, filter []string, t *testing.T) {
	file, fileError := os.Open(path)
	if fileError != nil {
		panic(fileError.Error())
	}
	defer file.Close()
	r := bufio.NewReader(file)
	showAll := false
	debugProcess, once, checkTerminated := false, true, false
	for {
		data, _, _ := r.ReadLine()
		if data != nil && len(string(data)) > 0 {
			line := string(data)
			*state.lastLine = line
			if _, found := passingFilter(line, filter); found {
				processLog(line, state, t)
			} else if *state.debug || showAll {
				log := "(DEBUG) " + state.id + " ::: " + line
				if t != nil {
					t.Log(log)
				} else {
					fmt.Println(log)
				}
			}

		}

		// if finishedTests(state.test.expectedEvents, *state.results) {
		// 	state.finished = true
		// }

		if state.cmd.Process != nil {
			pid := state.cmd.Process.Pid
			if checkTerminated && !*state.finished && state.requireAlive && !isPidAlive(pid) {
				log := state.id + " ::: PROCESS HAS TERMINATED ::: "
				(state.results)["TERMINATED"] = []*TestResult{{
					eventID: "TERMINATED",
					found:   true,
					passed:  false,
					line:    "Exiting Test",
					err:     fmt.Errorf(log),
					parent:  state.id,
					failNow: true,
				}}
				if t != nil {
					t.Logf(log)
				} else {
					println(log)
				}
				break
			}
			if debugProcess && once {
				println(" ::: +++++++++++++++++++ started process "+state.id+" PID: ", pid)
				once = false
			}
		}

		if *state.finished || *state.failed {
			break
		}
	}
	fmt.Printf(" ::: exiting %s ::: \n", state.id)
	*state.failed = true
	for _, dep := range state.depending {
		if dep != nil {
			*dep.finished = true
		}
	}
	*state.finished = true
}

func processLog(line string, state State, t *testing.T) {
	parent := state.id
	foundAwating := false
	for key, await := range state.awaiting {
		if foundAwating {
			continue
		} else if testRes := await.f(line); testRes.found {
			*state.awaiting[key].done = true
			*state.awaiting[key].pass = testRes.passed
			foundAwating = true
			if testRes.failNow {
				println("1111111111111111")
				*state.failed = true
				*state.finished = true
			}
		}
	}
	runtype := ""
	if state.testing {
		runtype = "!!! testing !!! "
		foundExpected := false
		results := state.results
		failed := false
		fail_now := false
		for _, event := range state.test.expectedEvents {
			//TODO: match with regex
			if strings.Contains(line, event) {
				if _, ok := results[event]; !ok {
					results[event] = []*TestResult{}
				} else {
					// reacurring event
				}
				if test, test_exists := state.test.tests[event]; test_exists {
					testRes := test(LogLine{line: line, parent: state.id})
					if testRes.failNow {
						fail_now = true
					}
					res, err := testRes.passed, testRes.err
					results[event] = append(results[event], &TestResult{event, true, res, line, err, state.id, fail_now})
				} else {
					results[event] = append(results[event], &TestResult{event, true, false, line, fmt.Errorf("Test not implemented for event %s", event), state.id, fail_now})
					failed = true
				}
				foundExpected = true
			}
			if foundExpected {
				continue
			}
		}
		pre := "[-] "
		if foundExpected {
			pre = "[+] "
		} else {
			foundUnexpected := false
			for _, event := range state.test.unexpectedEvents {
				if foundUnexpected {
					continue
				}
				//TODO: match with regex
				if strings.Contains(line, event) {
					if _, ok := results[event]; !ok { // first time event
						results[event] = []*TestResult{}
					}
					if test, test_exists := state.test.tests[event]; test_exists {
						testRes := test(LogLine{line: line, parent: state.id})
						if testRes.failNow {
							fail_now = true
						}
						res, err := testRes.passed, testRes.err
						if !res && state.test.strict {
							failed = true
							fail_now = true
						}
						results[event] = append(results[event], &TestResult{event, true, res, line, err, state.id, fail_now})
					} else {
						results[event] = append(results[event], &TestResult{event, true, false, line, fmt.Errorf("Unexpected event %s ::: %s", event, line), state.id, fail_now})
						if state.test.strict {
							failed = true
						}
					}
					foundUnexpected = true
					pre = "[X] "
				}
			}
		}
		log := parent + " ::: " + runtype + pre + line
		if *state.stdout {
			fmt.Println(log)
		}
		if fail_now {
			*state.failed = true
			*state.finished = true
			t.FailNow()
		}
		if failed {
			t.Fail()
			if state.test.strict {
				println(" ::: FAILED TEST WITH STRICT CONDITION ::: EXITING " + state.id + " ::: ")
				*state.finished = true
				*state.failed = true
				for _, dep := range state.depending {
					if dep != nil {
						println(" ::: FAILED TEST WITH STRICT CONDITION ::: EXITING " + dep.id + " ::: ")
						*dep.finished = true
					}
				}
				t.FailNow()
			} else {

			}
		}
	} else {
		log := parent + " ::: " + runtype + line
		fmt.Println(log)

	}
}

func printTestResult(res *TestResult, t *testing.T) {
	status := "FAILED"
	err := res.err
	if res.passed {
		status = "PASSED"
	}
	if err == nil {
		err = fmt.Errorf("")
	}
	line := res.line
	errMsg := err.Error()
	log := fmt.Sprintf("%s ::: %s ::: %s ::: %s ::: %s", status, res.parent, res.eventID, line, errMsg)
	fmt.Println(log)
}

func LogProcess(cmd CMD, t *testing.T, states *[]State) State {
	newState := logProcess(t, cmd.stateID, cmd.homepath, cmd.cmd, cmd.filter, cmd.testing, cmd.test, cmd.results, cmd.dep, cmd.failed, cmd.requireAlive, cmd.debug, cmd.stdout)
	*states = append(*states, newState)
	return newState
}

func logProcess(t *testing.T, id string, home string, cmd string, filter []string, testing bool, test TestProc, results map[string][]*TestResult, depends *State, failed *bool, requireAlive bool, debug *bool, stdout *bool) State {
	finished := false
	lastLine := "init"
	state := State{
		id:           id,
		finished:     &finished,
		awaiting:     map[string]Await{},
		testing:      testing,
		test:         test,
		results:      results,
		depending:    []*State{depends},
		cmd:          nil,
		failed:       failed,
		requireAlive: requireAlive,
		lastLine:     &lastLine,
		debug:        debug,
		stdout:       stdout,
	}
	os.Chdir(home)
	logFile := id + ".log"
	logPath := resetLog(home, logFile, "testutil/e2e/logs/")

	end := " 2>&1"
	full := "cd " + home + " && " + cmd + " >> " + logPath + end
	fullCMD := exec.Command("sh", "-c", full)
	state.cmd = fullCMD

	go fullCMD.Start()
	go readFile(logPath, state, filter, t)

	return state
}

func finalizeResults(t *testing.T) []*TestResult {

	fmt.Println(string("================================================="))
	fmt.Println(string("================ TEST DONE! ====================="))
	fmt.Println(string("================================================="))
	fmt.Println(string("=============== Expected Events ================="))

	finalExpectedTests := []*TestResult{}
	finalUnexpectedTests := []*TestResult{}
	count := 1
	for _, expected := range getExpectedEvents(states) {
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

	exit(states)
	final := append(finalExpectedTests, finalUnexpectedTests...)
	return final
}

func prepTest(t *testing.T) {
	tt = t
	if t != nil {
		t.Logf(" ::: Test Homepath ::: %s", homepath)
	}

	*failed = false
	states = []State{}
	results = map[string][]*TestResult{}

}

func wrapGoTest(t *testing.T, finalresults []*TestResult, err error) {
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

func exit(states []State) bool {
	processList := []*os.Process{}
	for _, state := range states {
		*state.finished = true
		if state.cmd.Process != nil {
			processList = append(processList, state.cmd.Process)
		}
	}
	sleep(2)
	killProcessess := false
	if killProcessess {
		for _, process := range processList {
			killPid(process.Pid)
			process.Kill()
		}
	}
	ExitLavaProcess()
	return true
}
