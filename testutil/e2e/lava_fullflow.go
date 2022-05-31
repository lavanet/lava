//file_pipe.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

// func finishedTests(expectedEvents []string, results map[string][]TestResult) bool {
// 	for _, eventID := range expectedEvents {
// 		if _, ok := results[eventID]; !ok {
// 			return false
// 		}
// 	}
// 	return true
// }

// func readFile(path string, done *bool, doneF func(string) bool) {
// func readFile(path string, state State, doneF func(string) bool) {

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
	// homepath := getHomePath() + "go/lava/"
	homepath := getHomePath() + "go/lava/"
	t.Logf("!!!!!!!!!!!! HOME !!!!!!!!!!!!!!! %s", homepath)

	// Test Flow
	node := LogProcess(CMD{
		stateID:      "starport",
		homepath:     homepath,
		cmd:          "killall starport; cd " + homepath + " && starport chain serve -v -r ",
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
		cmd:          "go run relayer/cmd/relayer/main.go test_client ETH1 jsonrpc --from user1",
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
	if *failed && t != nil {
		t.Errorf(" ::: Test Failed ::: ")
		t.FailNow()
	}
	// nothing([]State{init, client}) // we are not using client or init atm so theres a "declared but not used" error
	exit(states)
	final := append(finalExpectedTests, finalUnexpectedTests...)
	return final, nil
	// TODO: exit cleanly
}

func main() {
	// mainB() // when piping into go i.e - 6; cd ~/go/lava && starport chain serve -v -r |& go run x_test/lava_pipe.go
	FullFlowTest(nil)
}
