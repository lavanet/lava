package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

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
						if len(last) > 120 {
							last = last[:120] + "..."
						}
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
	debugProcess, once, checkTerminated := false, true, false
	for {
		data, _, err := r.ReadLine()
		if err != nil {
			// fmt.Println(string("XXX error XXX ") + string(err.Error()))
		} else {
			line := string(data)
			*state.lastLine = line
			if _, found := passingFilter(line, filter); found {
				processLog(line, state, t)
			} else if state.debug {
				log := "(DEBUG) " + state.id + " ::: " + line
				if t != nil {
					t.Log(log)
				} else {
					fmt.Println(log)
				}
			} else {
				// fmt.Println(parent + " ::: " + string("!!! nice !!! ") + string(line))
			}

		}
		if err == io.EOF {
			// fmt.Printf("EOF.")
			// break
		}

		// if finishedTests(state.test.expectedEvents, *state.results) {
		// 	state.finished = true
		// }

		if state.cmd.Process != nil {
			pid := state.cmd.Process.Pid
			if checkTerminated && !*state.finished && state.requireAlive && !isPidAlive(pid) {
				log := state.id + " ::: PROCESS HAS TERMINATED ::: "
				(*state.results)["TERMINATED"] = []TestResult{{
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
	for _, dep := range *state.depending {
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
		results := *state.results
		failed := false
		fail_now := false
		for _, event := range state.test.expectedEvents {
			//TODO: match with regex
			if strings.Contains(line, event) {
				if _, ok := results[event]; !ok {
					results[event] = []TestResult{}
				} else {
					// reacurring event
				}
				if test, test_exists := state.test.tests[event]; test_exists {
					testRes := test(LogLine{line: line, parent: state.id})
					if testRes.failNow {
						fail_now = true
					}
					res, err := testRes.passed, testRes.err
					results[event] = append(results[event], TestResult{event, true, res, line, err, state.id, fail_now})
				} else {
					results[event] = append(results[event], TestResult{event, true, false, line, fmt.Errorf("Test not implemented for event %s", event), state.id, fail_now})
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
					if _, ok := results[event]; !ok {
						results[event] = []TestResult{}
					} else {
						// reacurring event
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
						results[event] = append(results[event], TestResult{event, true, res, line, err, state.id, fail_now})
					} else {
						results[event] = append(results[event], TestResult{event, true, false, line, fmt.Errorf("Unexpected event %s ::: %s", event, line), state.id, fail_now})
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
		fmt.Println(log)
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
				for _, dep := range *state.depending {
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

func printTestResult(res TestResult, t *testing.T) {
	status := "FAILED"
	err := res.err
	if res.passed {
		status = "PASSED"
	}
	if err == nil {
		err = fmt.Errorf("")
	}
	short := 40 * 4
	line := res.line
	if len(line) > short {
		line = line[:short] + "..."
	}
	errMsg := err.Error()
	if len(errMsg) > short {
		errMsg = errMsg[:short] + "..."
	}
	log := fmt.Sprintf("%s ::: %s ::: %s ::: %s ::: %s", status, res.parent, res.eventID, line, errMsg)
	fmt.Println(log)
}

func LogProcess(cmd CMD, t *testing.T, states *[]State) State {
	newState := logProcess(t, cmd.stateID, cmd.homepath, cmd.cmd, cmd.filter, cmd.testing, cmd.test, cmd.results, cmd.dep, cmd.failed, cmd.requireAlive, cmd.debug)
	*states = append(*states, newState)
	return newState
}

func logProcess(t *testing.T, id string, home string, cmd string, filter []string, testing bool, test TestProcess, results *map[string][]TestResult, depends *State, failed *bool, requireAlive bool, debug bool) State {
	finished := false
	lastLine := "init"
	state := State{
		id:           id,
		finished:     &finished,
		awaiting:     map[string]Await{},
		testing:      testing,
		test:         test,
		results:      results,
		depending:    &[]*State{depends},
		cmd:          nil,
		failed:       failed,
		requireAlive: requireAlive,
		lastLine:     &lastLine,
		debug:        debug,
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
