//file_pipe.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func finishedTests(expectedEvents []string, results map[string][]TestResult) bool {
	for _, eventID := range expectedEvents {
		if _, ok := results[eventID]; !ok {
			return false
		}
	}
	return true
}

// func readFile(path string, done *bool, doneF func(string) bool) {
// func readFile(path string, state State, doneF func(string) bool) {
func readFile(path string, state State, filter []string, t *testing.T) {
	file, fileError := os.Open(path)
	if fileError != nil {
		panic(fileError.Error())
	}
	defer file.Close()
	r := bufio.NewReader(file)
	debugProcess, once := false, true
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
			if !*state.finished && state.requireAlive && !isPidAlive(pid) {
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

func passingFilter(line string, options []string) (string, bool) {
	if len(options) == 0 {
		return "", true
	}
	for _, key := range options {
		if strings.Contains(line, key) {
			return key, true
		}
	}
	return "", false
}

func getKeys(m map[string]Await) []string {
	j := 0
	keys := make([]string, len(m))
	for k := range m {
		keys[j] = k
		j++
	}
	return keys
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
func ExampleCmd_CombinedOutputX() {
	cmd := exec.Command("sh", "-c", "echo stdout; echo 1>&2 stderr")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Errorf(err.Error())
	}
	fmt.Printf("%s\n", stdoutStderr)
}
func ExitLavaProcess() {
	// print("XXXXXXXXXXXXXXXXXxx", pid)
	cmd := exec.Command("sh", "-c", "killall lavad ; killall starport ; killall main ;")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Errorf(err.Error())
	}
	fmt.Printf(" ::: Exiting Lava Process ::: %s\n", stdoutStderr)
	// return true
}
func killPid(pid int) bool {
	// print("XXXXXXXXXXXXXXXXXxx", pid)
	cmd := exec.Command("sh", "-c", "kill -9 "+fmt.Sprint(pid))
	// cmd := exec.Command("kill", "-9", fmt.Sprint(pid))
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Errorf(err.Error())
	}
	fmt.Printf(" ::: XXXXX Killed Process %d %s\n", pid, stdoutStderr)
	return true
}
func isPidAlive(pid int) bool {
	// print("XXXXXXXXXXXXXXXXXxx")
	cmd := exec.Command("sh", "-c", "ps -a | grep "+fmt.Sprint(pid))
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	res := fmt.Sprintf("%s", stdoutStderr)
	// fmt.Println(len(res), res)
	// print("XXXXXXXXXXXXXXXXXxx")
	if strings.Contains(res, "defunct") {
		return false
	}
	return true
}

type Event struct {
	id      string      `json:"id"`
	content string      `json:"content"`
	parent  string      `json:"parent"`
	extra   interface{} `json:"extra,omitempty"`
}
type TestResult struct {
	eventID string `json:"eventID"`
	// event   Event  `json:"eventID"`
	found  bool   `json:"eventID"`
	passed bool   `json:"passed"`
	line   string `json:"comment"`
	err    error  `json:"err"`
}
type State struct {
	id           string                   `json:"id"`
	finished     *bool                    `json:"finished"`
	awaiting     map[string]Await         `json:"awating"`
	testing      bool                     `json:"testing"`
	test         Test                     `json:"test"`
	results      *map[string][]TestResult `json:"results"`
	depending    *[]*State                `json:"depending"`
	cmd          *exec.Cmd                `json:"cmd"`
	failed       *bool                    `json:"failed"`
	requireAlive bool                     `json:"failed"`
}

type Await struct {
	done *bool                     `json:"done"`
	pass *bool                     `json:"pass"`
	f    func(string) (bool, bool) `json:"f"`
	msg  string                    `json:"msg"`
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
	logPath := resetLog(home, logFile, "testutil/e2e/logs/")
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

func resetLog(home string, logFile string, folder string) string {
	os.Chdir(home)
	if err := os.MkdirAll(home+folder, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	logPath := home + folder + logFile
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	return logPath
	// println("DDDDDDDONEEEEE RESET LOG")
}
func getHomePath() string {
	userHome, err := os.UserHomeDir()
	if err != nil {
		panic(err.Error())
	}
	return userHome + "/"
}

func raw_log(line string) (bool, bool) {
	contains := "raw_log"
	return advanceFlow(line, contains), true
}
func providers_ready(line string) (bool, bool) {
	contains := "lava_client_stake"
	return advanceFlow(line, contains), true
}

func found_rpc_reply(line string) (bool, bool) {
	contains := "reply JSONRPC_"
	return advanceFlow(line, contains), true
}
func found_relay_payment(line string) (bool, bool) {
	contains := "lava_relay_payment"
	return advanceFlow(line, contains), true
}
func node_reset(line string) (bool, bool) {
	contains := "🔄"
	return advanceFlow(line, contains), true
}
func node_ready(line string) (bool, bool) {
	contains := "🌍 Token faucet: http"
	return advanceFlow(line, contains), true
}
func new_epoch(line string) (bool, bool) {
	contains := "lava_new_epoch"
	return advanceFlow(line, contains), true
}

func testStart(line string) (bool, bool) {
	contains := "🌍 Token faucet: http"
	return advanceFlow(line, contains), true // found, pass
}

func test0(line string) (bool, bool) {
	contains := "Cosmos SDK"
	return advanceFlow(line, contains), true // found, pass
}
func test_found_pass(line string) (bool, string, error) {
	return true, line, nil
}
func test_found_fail(line string) (bool, string, error) {
	return false, line, nil
}

func test_ERR_client_entries_pairing(line string) (pass bool, retline string, err error) {
	return true, line, fmt.Errorf("ERR_client_entries_pairing is unexpected but still passing to finish fullflow")
}
func test_start(line string) (bool, string, error) {
	return true, line, fmt.Errorf("🔄 is expected")
}
func test_start_fail(line string) (bool, string, error) {
	return false, line, fmt.Errorf("🔄 is not expected")
}
func advanceFlow(line string, contains string) bool {
	if strings.Contains(line, contains) {
		// println("DONEEEE " + contains)
		// println("DONEEEE " + contains)
		// println("DONEEEE " + contains)
		// println("DONEEEE " + contains)
		// println("DONEEEE " + contains)
		// println("DONEEEE " + contains)
		// println("DONEEEE " + contains)
		return true
	}
	return false
}

// run node -r
// wait new epoch node
// run init
// await lava_client_stake_new from init + timeout
// await new epoch from node
// run client
// await new_relay_payment from node
// await 10 secs without error
// get test results
// func await(done *bool, msg string, after string) {
// 	for !*done {
// 		time.Sleep(1 * time.Second)
// 		println(" ::: " + msg)
// 	}
// 	println("*********************************************")
// 	println(fmt.Sprintf("***************  %s   *********************", after))
// 	println("*********************************************")
// }
func sleep(t int, failed *bool) {
	if !*failed {
		for i := 1; i <= t; i++ {
			//    fmt.Println(i)
			if *failed {
				break
			}
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
}

func await(state State, id string, f func(string) (bool, bool), msg string) {
	done, pass := false, false
	state.awaiting[id] = Await{&done, &pass, f, msg}
	awaitState(state)
}

type Test struct {
	expectedEvents   []string                                        `json:"expectedEvents"`
	unexpectedEvents []string                                        `json:"unexpectedEvents"`
	tests            map[string](func(string) (bool, string, error)) `json:"tests"`
	strict           bool                                            `json:"strict"`
}

func exit(states []State) bool {
	processList := []*os.Process{}
	for _, state := range states {
		*state.finished = true
		if state.cmd.Process != nil {
			processList = append(processList, state.cmd.Process)
		}
	}
	f := false
	sleep(2, &f)
	// for _, process := range processList {
	// 	killPid(process.Pid)
	// 	process.Kill()
	// }
	ExitLavaProcess()
	return true
}

type CMD struct {
	stateID      string
	homepath     string
	cmd          string
	filter       []string
	testing      bool
	test         Test
	results      *map[string][]TestResult
	dep          *State
	failed       *bool
	requireAlive bool
}

func LogProcess(cmd CMD, t *testing.T, states *[]State) State {
	newState := logProcess(t, cmd.stateID, cmd.homepath, cmd.cmd, cmd.filter, cmd.testing, cmd.test, cmd.results, cmd.dep, cmd.failed, cmd.requireAlive)
	*states = append(*states, newState)
	return newState
}

func tests() map[string](func(string) (bool, string, error)) {
	tests := map[string](func(string) (bool, string, error)){
		"🔄": test_start,
		// "🔄":                          test_found_pass,
		"🌍":                          test_found_pass,
		"lava_spec_add":              test_found_pass,
		"lava_provider_stake_new":    test_found_pass,
		"lava_client_stake_new":      test_found_pass,
		"lava_relay_payment":         test_found_pass,
		"ERR_client_entries_pairing": test_ERR_client_entries_pairing,
		"update pairing list!":       test_found_pass,
		"Client pubkey":              test_found_pass,
		"no pairings available":      test_found_fail,
		"rpc error":                  test_found_pass,
		// "error":                      test_found_fail,
		"reply": test_found_pass,
	}
	return tests
}

// TODO:
// [-] merge main
// [-] refactor & clean for PR
// [-] improvements:
// 		[-] add steps to test results,
// 		[-] add pass/fail to step,
//		[-] steps & events in order,
//		[-] await timeout
// [-] logProcess providers
// [-] go-test output
// [-] dockerize
// [-] github actions CI/CD
func FullFlowTest(t *testing.T) ([]TestResult, error) {

	// Test Configs
	nodeTest := Test{
		expectedEvents:   []string{"🔄", "🌍", "lava_spec_add", "lava_provider_stake_new", "lava_client_stake_new", "lava_relay_payment"},
		unexpectedEvents: []string{"ERR_client_entries_pairing", "ERR"},
		tests:            tests(),
		strict:           false}
	initTest := Test{
		expectedEvents:   []string{},
		unexpectedEvents: []string{"Error"},
		tests:            tests(),
		strict:           true}
	clientTest := Test{
		expectedEvents:   []string{"update pairing list!", "Client pubkey"},
		unexpectedEvents: []string{"no pairings available", "error", "Error", "signal: interrupt"},
		tests:            tests(),
		strict:           true}
	testfailed := false
	failed := &testfailed
	states := []State{}
	results := map[string][]TestResult{}
	homepath := getHomePath() + "go/lava/"

	// Test Flow
	node := LogProcess(CMD{
		stateID:      "starport",
		homepath:     homepath,
		cmd:          "killall starport; starport chain serve -v -r ",
		filter:       []string{"STARPORT]", "!", "lava_", "ERR_", "panic"},
		testing:      true,
		test:         nodeTest,
		results:      &results,
		dep:          nil,
		failed:       failed,
		requireAlive: true}, t, &states)
	await(node, "node reset", node_reset, "awating for node reset to proceed...")
	await(node, "node connected", node_ready, "awating for node api to proceed...")
	await(node, "node ready", new_epoch, "awating for new epoch to proceed...")
	// sleep(2, failed)

	init := LogProcess(CMD{
		stateID:      "init",
		homepath:     homepath,
		cmd:          "./init_chain_commands.sh",
		filter:       []string{"raw_log", "Error", "error", "panic"},
		testing:      true,
		test:         initTest,
		results:      &results,
		dep:          &node,
		failed:       failed,
		requireAlive: false}, t, &states)
	await(init, "get raw_log from init", raw_log, "awating for raw_log to proceed...")
	await(node, "init chain - providers ready", providers_ready, "awating for providers to proceed...")
	sleep(2, failed)

	client := LogProcess(CMD{
		stateID:      "client",
		homepath:     homepath,
		cmd:          "go run /home/magic/go/lava/relayer/cmd/relayer/main.go test_client ETH1 jsonrpc --from user1",
		filter:       []string{"reply", "no pairings available", "update", "connect", "rpc", "pubkey", "signal", "Error", "error", "panic"},
		testing:      true,
		test:         clientTest,
		results:      &results,
		dep:          &node,
		failed:       failed,
		requireAlive: false}, t, &states)
	await(client, "reply rpc", found_rpc_reply, "awating for rpc relpy to proceed...")
	await(node, "relay payment 1", found_relay_payment, "awating for FIRST payment to proceed...")
	println(" ::: GOT FIRST PAYMENT !!!")

	println("::::::::::::::::::::::::::::::::::::::::::::::")
	awaitErrorsTimeout := 10
	println(" ::: wait ", awaitErrorsTimeout, " seconds for potential errors...")
	sleep(awaitErrorsTimeout, failed)

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
	for _, expected := range append(nodeTest.expectedEvents, append(initTest.expectedEvents, clientTest.expectedEvents...)...) {
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
	if *failed {
		t.Errorf(" ::: Test Failed ::: ")
		t.FailNow()
	}
	// nothing([]State{init, client}) // we are not using client or init atm so theres a "declared but not used" error
	exit(states)
	final := append(finalExpectedTests, finalUnexpectedTests...)
	return final, nil
	// TODO: exit cleanly
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

func main() {
	// mainB() // when piping into go i.e - 6; cd ~/go/lava && starport chain serve -v -r |& go run x_test/lava_pipe.go
	FullFlowTest(nil)
}
