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
					// println(" ::: "+state.id+" ::: "+await.msg, *state.finished)
					if !*state.failed {
						println(" ::: " + state.id + " ::: " + await.msg)
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

func await(state State, id string, f func(string) (bool, bool), msg string) {
	done, pass := false, false
	state.awaiting[id] = Await{&done, &pass, f, msg}
	awaitState(state)
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
		// line, err := r.ReadString('\n')

		data, _, err := r.ReadLine()
		// num += 1
		if err != nil {
			// fmt.Println(string("XXX error XXX ") + string(err.Error()))
		} else {
			line := string(data)
			if _, found := passingFilter(line, filter); found {
				processLog(line, state, t)
			} else {
				// fmt.Println(parent + " ::: " + string("!!! nice !!! ") + string(line))
			}

			// if processLog(line, path, doneF) {
			// 	*done = true
			// }
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
				// require := fmt.Sprintf(" @@@@@@@@", state.id, state.requireAlive)
				log := state.id + " ::: PROCESS HAS TERMINATED ::: "
				(*state.results)["TERMINATED"] = []TestResult{TestResult{
					eventID: "TERMINATED",
					found:   true,
					passed:  false,
					line:    "Exiting Test",
					err:     fmt.Errorf(log),
				}}
				if t != nil {
					t.Logf(log)
				} else {
					println(log)
				}
				break
			}
			// process, err := os.FindProcess(int(pid))
			if debugProcess && once {
				println(" ::: +++++++++++++++++++ started process "+state.id+" PID: ", pid)
				once = false
			}
			// if err != nil {
			// 	println("RRRRRRRRRRRRRRRRRRRRRRR not found")
			// 	// fmt.Printf("Failed to find process: %s\n", err)
			// } else {
			// 	err := process.Signal(syscall.Signal(0))
			// 	// println(".....", pid, err)
			// 	if err != nil {
			// 		println("RRRRRRRRRRRRRRRRRRRRRRR process terminated")
			// 		break
			// 	}
			// 	// fmt.Printf("process.Signal on pid %d returned: %v\n", pid, err)
			// }
		} else {
			// println(".....") // process not started yet
		}
		if *state.finished || *state.failed {
			break
		} else {
			// print("not finished...")
		}

	}
	fmt.Println(fmt.Sprintf(" ::: exiting %s ::: ", state.id))
	*state.failed = true
	for _, dep := range *state.depending {
		if dep != nil {
			// println("!!!!!!!", dep.id, *dep.finished)
			*dep.finished = true
			// *(*dep).finished = true
			// println("!!!!!!!", dep.id, *dep.finished)
		}
	}
	*state.finished = true
	// println("XXXXXXXXXXXXX")
}

// func processLog(line string, parent string, doneF func(string) bool) bool {
func processLog(line string, state State, t *testing.T) {
	parent := state.id
	foundAwating := false
	for key, await := range state.awaiting {
		if foundAwating {
			continue
		} else if found, pass := await.f(line); found {
			*state.awaiting[key].done = true
			*state.awaiting[key].pass = pass
			foundAwating = true
		}
	}
	runtype := ""
	if state.testing {
		runtype = "!!! testing !!! "

		foundExpected := false
		results := *state.results
		failed := false
		for _, event := range state.test.expectedEvents {
			//TODO: match with regex
			if strings.Contains(line, event) {
				//TODO: match with regex
				if _, ok := results[event]; !ok {
					results[event] = []TestResult{}
				} else {
					// reacurring event
				}
				if test, test_exists := state.test.tests[event]; test_exists {
					res, _, err := test(line)
					results[event] = append(results[event], TestResult{event, true, res, line, err})
					// foundExpected = true
				} else {
					results[event] = append(results[event], TestResult{event, true, false, line, fmt.Errorf("Test not implemented for event %s", event)})
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
				//TODO: match with regex
				if foundUnexpected {
					continue
				}
				if strings.Contains(line, event) {
					//TODO: match with regex
					if _, ok := results[event]; !ok {
						results[event] = []TestResult{}
					} else {
						// reacurring event
					}
					if test, test_exists := state.test.tests[event]; test_exists {
						res, _, err := test(line)
						results[event] = append(results[event], TestResult{event, true, res, line, err})
						// foundExpected = true
						if !res && state.test.strict {
							failed = true
						}
					} else {
						results[event] = append(results[event], TestResult{event, true, false, line, fmt.Errorf("Unexpected event %s ::: %s", event, line)})
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
		if failed {
			// println(state.id + " XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
			t.Fail()
			if state.test.strict {
				println(" ::: FAILED TEST WITH STRICT CONDITION ::: EXITING " + state.id + " ::: ")
				*state.finished = true
				*state.failed = true
				for _, dep := range *state.depending {
					if dep != nil {
						println(" ::: FAILED TEST WITH STRICT CONDITION ::: EXITING " + dep.id + " ::: ")
						// println("!!!!!!!", dep.id, *dep.finished)
						*dep.finished = true
						// *(*dep).finished = true
						// println("!!!!!!!", dep.id, *dep.finished)
					}
				}
				t.FailNow()
			} else {
				// println("33333333222222XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")

			}
		}
		// if t != nil {
		// 	t.Log(log)
		// }
	} else {
		log := parent + " ::: " + runtype + line
		fmt.Println(log)
		// if t != nil {
		// 	t.Log(log)
		// }
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
	// fmt.Println(fmt.Sprintf("%s ::: %s ::: %s ::: %s", status, res.line))
	log := fmt.Sprintf("%s ::: %s ::: %s ::: %s", status, res.eventID, line, errMsg)
	fmt.Println(log)
	// if t != nil {
	// 	t.Log(log)
	// }
}

func LogProcess(cmd CMD, t *testing.T, states *[]State) State {
	newState := logProcess(t, cmd.stateID, cmd.homepath, cmd.cmd, cmd.filter, cmd.testing, cmd.test, cmd.results, cmd.dep, cmd.failed, cmd.requireAlive)
	*states = append(*states, newState)
	return newState
}

func logProcess(t *testing.T, id string, home string, cmd string, filter []string, testing bool, test Test, results *map[string][]TestResult, depends *State, failed *bool, requireAlive bool) State {

	finished := false
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
	}
	// state := State{id, &finished, map[string]Await{}, testing, test, results, &[]*State{depends}}
	// d := false
	// done := &d
	os.Chdir(home)
	logFile := id + ".log"
	// logPath := resetLog(home, logFile, "x_test/tests/integration/")
	// logPath := resetLog(home, logFile, "testutil/e2e/logs/")
	logPath := resetLog(home, logFile, "logs/")
	// logPath := home + logFile
	end := " 2>&1"
	// println("home", home)
	full := "cd " + home + " && " + cmd + " >> " + logPath + end
	// println("full", full)
	fullCMD := exec.Command("sh", "-c", full)
	state.cmd = fullCMD
	// fullCMD := exec.Command(cmd + end + " >> " + logPath + ")")
	go fullCMD.Start()
	go readFile(logPath, state, filter, t)
	//TODO: add cmd to state and kill when finished
	//https://stackoverflow.com/questions/11886531/terminating-a-process-started-with-os-exec-in-golang
	return state
}
