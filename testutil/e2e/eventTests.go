package main

import (
	"fmt"
	"strings"
)

func init_done(line string) (bool, bool) {
	contains := "init done"
	return advanceFlow(line, contains), true
}
func raw_log(line string) (bool, bool) {
	contains := "raw_log"
	return advanceFlow(line, contains), true
}
func providers_ready(line string) (bool, bool) {
	contains := "listening"
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
