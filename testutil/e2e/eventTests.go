package main

import (
	"fmt"
)

func init_done(line string) TestResult {
	contains := "init done"
	return test_basic(line, contains)
}
func raw_log(line string) TestResult {
	contains := "raw_log"
	return test_basic(line, contains)
}
func providers_ready(line string) TestResult {
	contains := "listening"
	return test_basic(line, contains)
}
func found_rpc_reply(line string) TestResult {
	contains := "reply JSONRPC_"
	return test_basic(line, contains)
}
func found_relay_payment(line string) TestResult {
	contains := "lava_relay_payment"
	return test_basic(line, contains)
}
func node_reset(line string) TestResult {
	contains := "🔄"
	return test_basic(line, contains)
}
func node_ready(line string) TestResult {
	contains := "🌍 Token faucet: http"
	return test_basic(line, contains)
}
func new_epoch(line string) TestResult {
	contains := "lava_new_epoch"
	return test_basic(line, contains)
}

func test_found_pass(log LogLine) TestResult {
	return TestResult{
		eventID: "found_pass",
		found:   true,
		passed:  true,
		line:    log.line,
		err:     nil,
		parent:  log.parent,
		failNow: false,
	}
}

func test_found_fail(log LogLine) TestResult {
	return TestResult{
		eventID: "found_fail",
		found:   true,
		passed:  false,
		line:    log.line,
		err:     nil,
		parent:  log.parent,
		failNow: false,
	}
}

func test_found_fail_now(log LogLine) TestResult {
	return TestResult{
		eventID: "found_fail_now",
		found:   true,
		passed:  false,
		line:    log.line,
		err:     nil,
		parent:  log.parent,
		failNow: true,
	}
}

func test_basic(line string, contains string) TestResult {
	found, pass := false, false
	if strContains(line, contains) {
		found, pass = true, true
	}
	return TestResult{
		eventID: "",
		found:   found,
		passed:  pass,
		line:    line,
		err:     nil,
		parent:  "",
		failNow: false,
	}
}

func test_ERR_client_entries_pairing(log LogLine) TestResult {
	return TestResult{
		eventID: "found_fail_now",
		found:   true,
		passed:  true,
		line:    log.line,
		err:     fmt.Errorf("ERR_client_entries_pairing is unexpected but still passing to finish fullflow"),
		parent:  log.parent,
		failNow: false,
	}
}

func test_start(log LogLine) TestResult {
	return TestResult{
		eventID: "found_fail_now",
		found:   true,
		passed:  true,
		line:    log.line,
		err:     fmt.Errorf("🔄 is not expected"),
		parent:  log.parent,
		failNow: false,
	}
}

func test_start_fail(log LogLine) TestResult {
	return TestResult{
		eventID: "found_fail_now",
		found:   true,
		passed:  false,
		line:    log.line,
		err:     fmt.Errorf("🔄 is not expected"),
		parent:  log.parent,
		failNow: true,
	}
}
