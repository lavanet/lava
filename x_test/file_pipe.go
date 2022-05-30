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
func readFile(path string, state State, filter []string) {
	file, fileError := os.Open(path)
	if fileError != nil {
		panic(fileError.Error())
	}
	defer file.Close()
	r := bufio.NewReader(file)
	for {
		// line, err := r.ReadString('\n')
		if *state.finished {
			break
		} else {
			// print("not finished...")
		}

		data, _, err := r.ReadLine()
		// num += 1
		if err != nil {
			// fmt.Println(string("XXX error XXX ") + string(err.Error()))
		} else {
			line := string(data)
			if _, found := passingFilter(line, filter); found {
				processLog(line, state)
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

	}
	println(" ::: exiting ", state.id)
	for _, dep := range *state.depending {
		if dep != nil {
			// println("!!!!!!!", dep.id, *dep.finished)
			*dep.finished = true
			// *(*dep).finished = true
			// println("!!!!!!!", dep.id, *dep.finished)
		}
	}
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
func processLog(line string, state State) {
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
						results[event] = append(results[event], TestResult{event, true, false, line, fmt.Errorf("Unexpected event %s", event)})
						if state.test.strict {
							failed = true
						}
					}
					foundUnexpected = true
					pre = "[X] "

				}
			}
		}
		if failed {
			// println("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
			if state.test.strict {
				// println("SSSSSSSSXXXXXXXXXXXXXx")
				*state.finished = true
			} else {
				// println("33333333222222XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")

			}
		}
		// fmt.Println("!!! testing !!! " + pre + line)
		fmt.Println(parent + " ::: " + runtype + pre + line)
	} else {
		fmt.Println(parent + " ::: " + runtype + string(line))
	}
}
func ExampleCmd_CombinedOutput() {
	cmd := exec.Command("sh", "-c", "echo stdout; echo 1>&2 stderr")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", stdoutStderr)
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
	id        string                   `json:"id"`
	finished  *bool                    `json:"finished"`
	awaiting  map[string]Await         `json:"awating"`
	testing   bool                     `json:"testing"`
	test      Test                     `json:"test"`
	results   *map[string][]TestResult `json:"results"`
	depending *[]*State                `json:"depending"`
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
	for !fin {
		if *state.finished {
			break
		}

		if len(state.awaiting) == 0 {
			fin = true
			break
		} else {
			time.Sleep(1 * time.Second)
			for key, await := range state.awaiting {
				// println(" ::: "+state.id+" ::: "+await.msg, *state.finished)
				println(" ::: " + state.id + " ::: " + await.msg)
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
	println("*********************************************")
	for _, key := range markForDelete {
		println(" ::: DONE ::: " + state.id + " ::: " + key)
	}
	println("*********************************************")

}

func logProcess(id string, home string, cmd string, filter []string, testing bool, test Test, results *map[string][]TestResult, depends *State) State {

	finished := false
	state := State{id, &finished, map[string]Await{}, testing, test, results, &[]*State{depends}}
	// d := false
	// done := &d
	os.Chdir(home)
	logFile := id + ".log"
	logPath := resetLog(home, logFile, "x_test/tests/integration/")
	// logPath := home + logFile
	end := " 2>&1"
	// println("home", home)
	full := "cd " + home + " && " + cmd + " >> " + logPath + end
	// println("full", full)
	fullCMD := exec.Command("sh", "-c", full)
	// fullCMD := exec.Command(cmd + end + " >> " + logPath + ")")
	go fullCMD.Start()
	go readFile(logPath, state, filter)
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

func advanceFlow_init(line string) (bool, bool) {
	contains := "lava_client_stake"
	return advanceFlow(line, contains), true
}

func advanceFlow_client(line string) (bool, bool) {
	contains := "lava_relay_payment"
	return advanceFlow(line, contains), true
}
func advanceFlow_node(line string) (bool, bool) {
	contains := "🌍 Token faucet: http"
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

func test_ERR_client_entries_pairing(line string) (bool, string, error) {
	return false, line, fmt.Errorf("ERR_client_entries_pairing should't happen")
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
func sleep(t int) {
	time.Sleep(time.Duration(t) * time.Second)
}

func await(state State, id string, f func(string) (bool, bool), msg string) {
	done, pass := false, false
	state.awaiting[id] = Await{&done, &pass, f, msg}
	awaitState(state)
}

func tests() map[string](func(string) (bool, string, error)) {
	tests := map[string](func(string) (bool, string, error)){
		"🔄":                          test_start,
		"🌍":                          test_found_pass,
		"lava_spec_add":              test_found_pass,
		"lava_provider_stake_new":    test_found_pass,
		"lava_client_stake_new":      test_found_pass,
		"lava_relay_payment":         test_found_pass,
		"ERR_client_entries_pairing": test_ERR_client_entries_pairing,
		"update pairing list!":       test_found_pass,
		"Client pubkey":              test_found_pass,
	}
	return tests
}

type Test struct {
	expectedEvents   []string                                        `json:"expectedEvents"`
	unexpectedEvents []string                                        `json:"unexpectedEvents"`
	tests            map[string](func(string) (bool, string, error)) `json:"tests"`
	strict           bool                                            `json:"strict"`
}

func exit(states []State) bool {
	for _, state := range states {
		*state.finished = true
	}
	sleep(1)
	return true
}

func fullTest() {
	// tests :=
	// expectedEvents := []string{"🔄", "🌍", "lava_spec_add", "lava_provider_stake_new", "lava_client_stake_new", "lava_relay_payment"}
	// expectedEvents := []string{"🌍", "lava_spec_add", "lava_provider_stake_new", "lava_client_stake_new", "lava_relay_payment"}
	expectedEvents := []string{"🔄", "🌍", "lava_spec_add", "lava_provider_stake_new", "lava_relay_payment"}
	unexpectedEvents := []string{"ERR_client_entries_pairing", "ERR"}
	// unexpectedEvents := []string{"🔄", "ERR_client_entries_pairing", "ERR"}
	nodeStrict := false
	nodeTest := Test{expectedEvents, unexpectedEvents, tests(), nodeStrict}
	initTest := Test{[]string{}, []string{"error", "Error"}, tests(), true}
	clientTest := Test{[]string{"update pairing list!", "Client pubkey"}, []string{"no pairings available", "error", "Error"}, tests(), true}
	results := map[string][]TestResult{}

	homepath := getHomePath() + "go/lava/"

	node := logProcess("starport", homepath, "killall starport; starport chain serve -v -r ", []string{"STARPORT]", "!", "lava_", "ERR_"}, true, nodeTest, &results, nil)
	await(node, "node ready", advanceFlow_node, "awating for node to proceed...")
	sleep(1)

	init := logProcess("init", homepath, "./init_chain_commands.sh", []string{"raw_log", "Error", "error"}, true, initTest, &results, &node)
	await(node, "init chain - providers ready", advanceFlow_init, "awating for providers to proceed...")

	client := logProcess("client", homepath, "go run /home/magic/go/lava/relayer/cmd/relayer/main.go test_client ETH1 jsonrpc --from user1", []string{"update", "connect", "reply", "pubkey", "Error", "error"}, true, clientTest, &results, &node)
	await(node, "relay payment 1", advanceFlow_client, "awating for FIRST payment to proceed...")
	println(" ::: GOT FIRST PAYMENT !!!")

	await(node, "relay payment 2", advanceFlow_client, "awating for SECOND payment to proceed...")
	println(" ::: GOT SECOND PAYMENT !!! ")

	println("::::::::::::::::::::::::::::::::::::::::::::::")
	awaitErrorsTimeout := 3
	println(" ::: wait ", awaitErrorsTimeout, " seconds for potential errors...")
	sleep(awaitErrorsTimeout)

	println("::::::::::::::::::::::::::::::::::::::::::::::")
	println("::::::::::::::::::::::::::::::::::::::::::::::")
	println("::::::::::::::::::::::::::::::::::::::::::::::")

	fmt.Println(string("================================================="))
	fmt.Println(string("================ TEST DONE! ====================="))
	fmt.Println(string("================================================="))
	fmt.Println(string("=============== Expected Events ================="))

	expectedTests := []TestResult{}
	count := 1
	for _, expected := range append(expectedEvents, clientTest.expectedEvents...) {
		// fmt.Println("XXX " + expected)
		if testList, foundEvent := results[expected]; foundEvent {
			// fmt.Println("foundxxxxx")
			for _, res := range testList {
				expectedTests = append(expectedTests, res)
				printTestResult(res)
				// fmt.Println("RRRRRRRRRRR")
				count += 1
			}
		}
		delete(results, expected)
	}

	count = 1
	fmt.Println(string("============== Unexpected Events ================"))
	for _, testList := range results {
		for _, res := range testList {
			count += 1
			printTestResult(res)
		}
	}
	fmt.Println(string("================================================="))
	exit([]State{node, init, client})
	// TODO: exit cleanly
}
func printTestResult(res TestResult) {
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
	fmt.Println(fmt.Sprintf("%s ::: %s ::: %s ::: %s", status, res.eventID, line, errMsg))
}
func main() {
	// mainB() // when piping into go i.e - 6; cd ~/go/lava && starport chain serve -v -r |& go run x_test/lava_pipe.go
	fullTest()
}
