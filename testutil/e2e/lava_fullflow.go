//file_pipe.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

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

func ExitLavaProcess() {
	cmd := exec.Command("sh", "-c", "killall lavad ; killall ignite ; killall starport ; killall main ; killall lavad")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Errorf(err.Error()).Error())
	}
	fmt.Printf(" ::: Exiting Lava Process ::: %s\n", stdoutStderr)
}

func tests() map[string]func(LogLine) TestResult {
	tests := map[string](func(LogLine) TestResult){
		"🔄":                          test_start,
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
		"reply":                      test_found_pass,
		"refused":                    test_found_fail,
		"listening":                  test_found_pass,
		"init done":                  test_found_pass,
		"connection refused":         test_found_fail_now,
		"cannot build app":           test_found_fail_now,
		"exit status":                test_found_fail_now,
	}
	return tests
}

// TODO:
// [+] merge main
// [+] refactor & clean for PR
// [-] improvements:
// 		[-] add steps to test results,
// 		[-] add pass/fail to step,
//		[-] steps & events in order,
//		[-] await timeout
//		[+] test timeout
// [+] logProcess providers
// [+] go-test output
// [-] dockerize
// [+] github actions CI/CD
func FullFlowTest(t *testing.T) ([]TestResult, error) {
	// Setup Env
	// homepath := "$LAVA"
	resetGenesis := true
	isGithubAction := false
	homepath := os.Getenv("LAVA")
	if homepath == "" {
		homepath = getHomePath()
		if strings.Contains(homepath, "runner") { // on github
			homepath += "work/lava/lava"
			isGithubAction = true
		} else {
			homepath += "go/lava" //local
		}
	}
	homepath += "/"

	if t != nil {
		t.Logf(" ::: Test Homepath ::: %s", homepath)
	}
	// lava_serve_cmd := "killall starport; cd " + homepath + " && starport chain serve -v -r  "
	lava_serve_cmd := "killall ignite; cd " + homepath + " && ignite chain serve -v -r  "
	usingLavad := false
	if !resetGenesis {
		lava_serve_cmd = "lavad start "
		usingLavad = true
	}

	// Test Configs
	nodeTest := TestProcess{
		expectedEvents:   []string{"🔄", "🌍", "lava_spec_add", "lava_provider_stake_new", "lava_client_stake_new", "lava_relay_payment"},
		unexpectedEvents: []string{"exit status", "cannot build app", "connection refused", "ERR_client_entries_pairing", "ERR"},
		tests:            tests(),
		strict:           false}
	initTest := TestProcess{
		expectedEvents:   []string{"init done"},
		unexpectedEvents: []string{"Error"},
		tests:            tests(),
		strict:           true}
	providersTest := TestProcess{
		expectedEvents:   []string{"listening"},
		unexpectedEvents: []string{"refused"},
		tests:            tests(),
		strict:           true}
	clientTest := TestProcess{
		expectedEvents:   []string{"update pairing list!", "Client pubkey"},
		unexpectedEvents: []string{"no pairings available", "Error", "signal: interrupt"},
		// unexpectedEvents: []string{"no pairings available", "error", "Error", "signal: interrupt"},
		tests:  tests(),
		strict: true}
	testfailed := false
	failed := &testfailed
	states := []State{}
	results := map[string][]TestResult{}

	// Test Full Flow
	node := LogProcess(CMD{
		stateID:      "ignite",
		homepath:     homepath,
		cmd:          lava_serve_cmd,
		filter:       []string{"STARPORT]", "!", "lava_", "ERR_", "panic"},
		testing:      true,
		test:         nodeTest,
		results:      &results,
		dep:          nil,
		failed:       failed,
		requireAlive: false,
		debug:        false,
	}, t, &states)
	// await(node, "node reset", node_reset, "awating for node reset to proceed...")
	if !usingLavad {
		await(node, "node connected", node_ready, "awating for node api to proceed...")
	}

	// await(node, "node ready", new_epoch, "awating for new epoch to proceed...")
	init_chain := true
	if init_chain {
		sleep(2, failed)
		if resetGenesis || isGithubAction {
			init := LogProcess(CMD{
				stateID:      "init",
				homepath:     homepath,
				cmd:          "$LAVA/.scripts/init.sh",
				filter:       []string{":::", "raw_log", "Error", "error", "panic"},
				testing:      true,
				test:         initTest,
				results:      &results,
				dep:          &node,
				failed:       failed,
				requireAlive: false,
				debug:        false}, t, &states)
			await(init, "get init done", init_done, "awating for init to proceed...")
		}
	}
	providerAsCMD := false
	if providerAsCMD {
		home := homepath
		os.Chdir(home)

		id := "provX"
		logFile := id + ".log"
		// logPath := resetLog(home, logFile, "x_test/tests/integration/")
		// logPath := resetLog(home, logFile, "testutil/e2e/logs/")
		logPath := resetLog(home, logFile, "logs/")
		// logPath := home + logFile
		end := " 2>&1"
		// println("home", home)
		cmd := "go run relayer/cmd/relayer/main.go server 127.0.0.1 2221 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer1"
		full := "cd " + home + " && " + cmd + " >> " + logPath + end
		// println("full", full)
		fullCMD := exec.Command("sh", "-c", full)

		// fullCMD := exec.Command(cmd + end + " >> " + logPath + ")")
		err := fullCMD.Run()
		t.Logf(" xxxxxxx ::: %s", err)
		// go readFile(logPath, state, filter, t)
	}

	run_providers_osmosis := true
	if run_providers_osmosis {
		println(" ::: Starting Providers Processes [Osmosis] ::: ")
		prov_osm := LogProcess(CMD{
			stateID:  "providers_osmosis",
			homepath: homepath,
			// cmd:          "$LAVA/providers_osmosis.sh",
			cmd:          "$LAVA/.scripts/osmosis.sh", // with mock
			filter:       []string{"updated", "server", "error"},
			testing:      true,
			test:         providersTest,
			results:      &results,
			dep:          &node,
			failed:       failed,
			requireAlive: false,
			debug:        true}, t, &states)
		println(" ::: Providers Processes Started ::: ")
		await(prov_osm, "Osmosis providers ready", providers_ready, "awating for providers to listen to proceed...")
	}
	run_providers_manual := false
	if run_providers_manual {

		prov1 := LogProcess(CMD{
			stateID:      "provider1",
			homepath:     homepath,
			cmd:          "go run relayer/cmd/relayer/main.go server 127.0.0.1 2221 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer1",
			filter:       []string{"updated", "server", "error"},
			testing:      true,
			test:         providersTest,
			results:      &results,
			dep:          &node,
			failed:       failed,
			requireAlive: false,
			debug:        true}, t, &states)
		println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! ", prov1.id)
		LogProcess(CMD{
			stateID:      "provider2",
			homepath:     homepath,
			cmd:          "go run relayer/cmd/relayer/main.go server 127.0.0.1 2222 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer2",
			filter:       []string{"updated", "server", "error"},
			testing:      true,
			test:         providersTest,
			results:      &results,
			dep:          &node,
			failed:       failed,
			requireAlive: false,
			debug:        true}, t, &states)
		println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! ")
		LogProcess(CMD{
			stateID:      "provider3",
			homepath:     homepath,
			cmd:          "go run relayer/cmd/relayer/main.go server 127.0.0.1 2223 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer3",
			filter:       []string{"updated", "server", "error"},
			testing:      true,
			test:         providersTest,
			results:      &results,
			dep:          &node,
			failed:       failed,
			requireAlive: false,
			debug:        true}, t, &states)
		println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! ")
		LogProcess(CMD{
			stateID:      "provider4",
			homepath:     homepath,
			cmd:          "go run relayer/cmd/relayer/main.go server 127.0.0.1 2224 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer4",
			filter:       []string{"updated", "server", "error"},
			testing:      true,
			test:         providersTest,
			results:      &results,
			dep:          &node,
			failed:       failed,
			requireAlive: false,
			debug:        true}, t, &states)
		println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! ")
		prov5 := LogProcess(CMD{
			stateID:      "provider5",
			homepath:     homepath,
			cmd:          "go run relayer/cmd/relayer/main.go server 127.0.0.1 2225 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer5",
			filter:       []string{"updated", "server", "error"},
			testing:      true,
			test:         providersTest,
			results:      &results,
			dep:          &node,
			failed:       failed,
			requireAlive: false,
			debug:        true}, t, &states)
		println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!! Providers")
		await(prov5, "providers ready", providers_ready, "awating for providers to proceed...")
	}

	run_client_osmosis := true
	if run_client_osmosis {
		clientOsmosis := LogProcess(CMD{
			stateID:      "clientOsmo",
			homepath:     homepath,
			cmd:          "go run " + homepath + "relayer/cmd/relayer/main.go test_client COS3 tendermintrpc --from user2 && echo \"::: osmosis finished 1 :::\"",
			filter:       []string{":::", "reply", "no pairings available", "update", "connect", "rpc", "pubkey", "signal", "Error", "error", "panic"},
			testing:      true,
			test:         clientTest,
			results:      &results,
			dep:          &node,
			failed:       failed,
			requireAlive: false,
			debug:        true}, t, &states)
		// sleep(15, failed)
		await(clientOsmosis, "osmosis client1 finished", osmosis_finished, "awating for osmosis1 to finish to proceed...")
		clientOsmosis2 := LogProcess(CMD{
			stateID:      "clientOsmo2",
			homepath:     homepath,
			cmd:          "go run " + homepath + "relayer/cmd/relayer/main.go test_client COS3 tendermintrpc --from user2 && echo \"::: osmosis finished 2 :::\"",
			filter:       []string{":::", "reply", "no pairings available", "update", "connect", "rpc", "pubkey", "signal", "Error", "error", "panic"},
			testing:      true,
			test:         clientTest,
			results:      &results,
			dep:          &node,
			failed:       failed,
			requireAlive: false,
			debug:        true}, t, &states)
		// await(clientOsmosis, "reply rpc", found_rpc_reply, "awating for rpc relpy to proceed...")
		// TODO: check relay payment is COS3
		await(node, "relay payment 2 osmosis", found_relay_payment, "awating for SECOND payment to proceed... "+clientOsmosis2.id)
		println(" ::: GOT OSMOSIS PAYMENT !!!")
	}
	run_providers_eth := true
	if run_providers_eth {
		println(" ::: Starting Providers Processes [ETH] ::: ")
		prov_eth := LogProcess(CMD{
			stateID:  "providers_eth",
			homepath: homepath,
			// cmd:          "$LAVA/providers_eth.sh",
			cmd:          "$LAVA/.scripts/eth.sh", // with mock
			filter:       []string{"updated", "server", "error"},
			testing:      true,
			test:         providersTest,
			results:      &results,
			dep:          &node,
			failed:       failed,
			requireAlive: false,
			debug:        true}, t, &states)
		println(" ::: Providers Processes Started ::: ")
		await(prov_eth, "ETH providers ready", providers_ready, "awating for providers to listen to proceed...")
	}
	run_client_eth := true
	if run_client_eth {
		sleep(1, failed)
		if !run_providers_eth {
			sleep(60, failed)
		}

		clientEth := LogProcess(CMD{
			stateID:      "clientEth",
			homepath:     homepath,
			cmd:          "go run " + homepath + "relayer/cmd/relayer/main.go test_client ETH1 jsonrpc --from user1",
			filter:       []string{"reply", "no pairings available", "update", "connect", "rpc", "pubkey", "signal", "Error", "error", "panic"},
			testing:      true,
			test:         clientTest,
			results:      &results,
			dep:          &node,
			failed:       failed,
			requireAlive: false,
			debug:        true}, t, &states)
		await(clientEth, "reply rpc", found_rpc_reply, "awating for rpc relpy to proceed...")
		await(node, "relay payment 1 eth", found_relay_payment, "awating for FIRST payment to proceed...")
		println(" ::: GOT FIRST PAYMENT !!!")
	}

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
	return final, nil
}

func main() {
	FullFlowTest(nil)
	// mainB() // when piping into go i.e - 6; cd ~/go/lava && ignite chain serve -v -r |& go run x_test/lava_pipe.go
}
