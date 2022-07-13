//file_pipe.go
package main

import (
	"fmt"
	"testing"
)

var nodeTest = TestProc{
	filter:           []string{"STARPORT]", "!", "lava_", "ERR_", "panic"},
	expectedEvents:   []string{"🌍", "lava_spec_add", "lava_provider_stake_new", "lava_client_stake_new", "lava_relay_payment"},
	unexpectedEvents: []string{"exit status", "cannot build app", "connection refused", "ERR_client_entries_pairing", "ERR"},
	tests:            events(),
	strict:           false}
var initTest = TestProc{
	filter:           []string{":::", "raw_log", "Error", "error", "panic"},
	expectedEvents:   []string{"init done"},
	unexpectedEvents: []string{"Error"},
	tests:            events(),
	strict:           true}
var providersTest = TestProc{
	filter:           []string{"Server", "updated", "server", "error"},
	expectedEvents:   []string{"listening"},
	unexpectedEvents: []string{"ERROR", "refused", "Missing Payment"},
	tests:            events(),
	strict:           true}
var clientTest = TestProc{
	filter:         []string{":::", "reply", "no pairings available", "update", "connect", "rpc", "pubkey", "signal", "Error", "error", "panic"},
	expectedEvents: []string{"update pairing list!", "Client pubkey"},
	// unexpectedEvents: []string{"no pairings available", "error", "Error", "signal: interrupt"},
	unexpectedEvents: []string{"no pairings available", "Error", "signal: interrupt"},
	tests:            events(),
	strict:           true}

func FullFlowTest(t *testing.T) ([]TestResult, error) {
	prepTest(t)

	// Test Configs
	resetGenesis := true
	init_chain := true
	run_providers_osmosis := true
	run_providers_eth := true
	run_client_osmosis := true
	run_client_eth := true

	start_lava := "killall ignite; killall lavad; cd " + homepath + " && ignite chain serve -v -r  "
	if !resetGenesis {
		start_lava = "lavad start "
	}

	// Start Test Processes
	node := TestProcess("ignite", start_lava, nodeTest)
	await(node, "lava node is running", lava_up, "awaiting for node to proceed...")

	if init_chain {
		sleep(2)
		init := TestProcess("init", homepath+"scripts/init.sh", initTest)
		await(init, "get init done", init_done, "awaiting for init to proceed...")
	}

	if run_providers_osmosis {
		fmt.Println(" ::: Starting Providers Processes [Osmosis] ::: ")
		prov_osm := TestProcess("providers_osmosis", homepath+"scripts/osmosis.sh", providersTest)
		println(" ::: Providers Processes Started ::: ")
		await(prov_osm, "Osmosis providers ready", providers_ready, "awaiting for providers to listen to proceed...")

		if run_client_osmosis {
			sleep(1)
			fmt.Println(" ::: Starting Client Process [ETH] ::: ")
			clientOsmo := TestProcess("clientOsmo", "lavad test_client COS3 tendermintrpc --from user2", clientTest)
			await(node, "relay payment 1/2 osmosis", found_relay_payment, "awaiting for OSMOSIS payment to proceed... ")
			fmt.Println(" ::: GOT OSMOSIS PAYMENT !!!")
			silent(clientOsmo)
			silent(prov_osm)
		}
	}
	if run_providers_eth {
		fmt.Println(" ::: Starting Providers Processes [ETH] ::: ")
		prov_eth := TestProcess("providers_eth", homepath+"scripts/eth.sh", providersTest)
		fmt.Println(" ::: Providers Processes Started ::: ")
		await(prov_eth, "ETH providers ready", providers_ready_eth, "awaiting for providers to listen to proceed...")

		if run_client_eth {
			sleep(1)
			fmt.Println(" ::: Starting Client Process [COS3] ::: ")
			clientEth := TestProcess("clientEth", "lavad test_client ETH1 jsonrpc --from user1", clientTest)
			await(clientEth, "reply rpc", found_rpc_reply, "awaiting for rpc reply to proceed...")
			await(node, "relay payment 2/2 eth", found_relay_payment, "awaiting for ETH payment to proceed...")
			fmt.Println(" ::: GOT ETH PAYMENT !!!")
			silent(clientEth)
			silent(prov_eth)
		}
	}

	// FINISHED TEST PROCESSESS
	println("::::::::::::::::::::::::::::::::::::::::::::::")
	awaitErrorsTimeout := 10
	fmt.Println(" ::: wait ", awaitErrorsTimeout, " seconds for potential errors...")
	sleep(awaitErrorsTimeout)
	fmt.Println("::::::::::::::::::::::::::::::::::::::::::::::")
	fmt.Println("::::::::::::::::::::::::::::::::::::::::::::::")
	fmt.Println("::::::::::::::::::::::::::::::::::::::::::::::")

	// Finalize & Display Results
	final := finalizeResults(t)

	return final, nil
}

func main() {
	FullFlowTest(nil)
}
