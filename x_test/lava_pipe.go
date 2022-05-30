//simple_pipe.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func test(x string) string {
	return "++++++++++++++++++++++++NICE " + x
}

func test_found_passX(line string) (bool, string, error) {
	return true, line, nil
}

func test_ERR_client_entries_pairingX(line string) (bool, string, error) {
	return false, line, fmt.Errorf("ERR_client_entries_pairing should't happen")
}
func test_startX(line string) (bool, string, error) {
	return true, line, fmt.Errorf("🔄 is expected")
}

type EventX struct {
	id      string      `json:"id"`
	content string      `json:"content"`
	parent  string      `json:"parent"`
	extra   interface{} `json:"extra,omitempty"`
}
type TestResultX struct {
	eventID string `json:"eventID"`
	// event   Event  `json:"eventID"`
	found  bool   `json:"eventID"`
	passed bool   `json:"passed"`
	line   string `json:"comment"`
	err    error  `json:"err"`
}

func finishedTestsX(expectedEvents []string, results map[string][]TestResult) bool {
	for _, eventID := range expectedEvents {
		if _, ok := results[eventID]; !ok {
			return false
		}
	}
	return true
}

func fullflow() {
	tests := map[string](func(string) (bool, string, error)){
		"🔄":                          test_start,
		"🌍":                          test_found_pass,
		"lava_spec_add":              test_found_pass,
		"lava_provider_stake_new":    test_found_pass,
		"lava_client_stake_new":      test_found_pass,
		"lava_relay_payment":         test_found_pass,
		"ERR_client_entries_pairing": test_ERR_client_entries_pairing,
	}
	expectedEvents := []string{"🔄", "🌍", "lava_spec_add", "lava_provider_stake_new", "lava_client_stake_new", "lava_relay_payment"}
	unexpectedEvents := []string{"ERR_client_entries_pairing", "ERR"}
	// expectedEvents := []string{"🌍", "lava_spec_add", "lava_provider_stake_new", "lava_client_stake_new", "lava_relay_payment"}
	// unexpectedEvents := []string{"ERR", "🌍", "🔄"}

	// expectedEvents := []string{"🔄", "🌍", "lava_spec_add", "lava_provider_stake_new", "lava_client_stake_new", "lava_relay_payment","ERR_client_entries_pairing"}
	// expectedEvents := []string{"🔄", "lava_spec_add", "lava_provider_stake_new", "lava_client_stake_new", "lava_relay_payment"}
	// expectedEvents := []string{"lava_relay_payment", "lava_spec_add", "lava_provider_stake_new", "ERR_client_entries_pairing"}
	// expectedEvents = append(expectedEvents, "ERR_client_entries_paring")
	// unexpectedEvents := []string{}

	// events := []string{"lava_relay_payment"}
	// events := []Event{{"lava_relay_payment",}}
	results := map[string][]TestResult{}
	strict := true
	r := bufio.NewReader(os.Stdin)
	for {
		line, err := r.ReadString('\n')
		line = strings.TrimSuffix(line, "\n")
		// buf =? buf[:]
		// if num > 3 {
		// 	os.Exit(0)
		// }
		//TODO: filterList
		failed := false
		if strings.Contains(line, "STARPORT]") || strings.Contains(line, "!") || strings.Contains(line, "lava_") || strings.Contains(line, "ERR_") {
			foundExpected := false
			for _, event := range expectedEvents {
				//TODO: match with regex
				if strings.Contains(line, event) {
					//TODO: match with regex
					if _, ok := results[event]; !ok {
						results[event] = []TestResult{}
					} else {
						// reacurring event
					}
					if test, test_exists := tests[event]; test_exists {
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
				for _, event := range unexpectedEvents {
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
						if test, test_exists := tests[event]; test_exists {
							res, _, err := test(line)
							results[event] = append(results[event], TestResult{event, true, res, line, err})
							// foundExpected = true
						} else {
							results[event] = append(results[event], TestResult{event, true, false, line, fmt.Errorf("Unexpected event %s", event)})
							if strict {
								failed = true
							}
						}
						foundUnexpected = true
						pre = "[X] "

					}
				}
			}
			fmt.Println("!!! testing !!! " + pre + line)
		}
		// num += 1
		if err != nil {
			fmt.Print(string("XXX error XXX ") + string(err.Error()))
		}

		if err == io.EOF {
			break
		}
		if finishedTestsX(expectedEvents, results) || (strict && failed) {
			break
		}
		//TODO: Set time out
	}

	for _, event := range expectedEvents {
		if _, ok := results[event]; !ok {
			results[event] = append(results[event], TestResult{event, false, false, fmt.Sprintf("Expected Event Not Found"), fmt.Errorf("expected event %s not found", event)})
		}
	}

	fmt.Println(string("================================================="))
	fmt.Println(string("================ TEST DONE! ====================="))
	fmt.Println(string("================================================="))
	fmt.Println(string("=============== Expected Events ================="))

	expectedTests := []TestResult{}
	count := 1
	for _, expected := range expectedEvents {
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

	// TODO: exit cleanly
}
func printTestResultX(res TestResult) {
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

//simple_pipe.go
func mainB() {
	fullflow()
	// nBytes, nChunks := int64(0), int64(0)
	// buf := make([]byte, 0, 4*1024)
	// num := 0

	// m := map[string]interface{}{}
	// f := map[string](func(string) string){}
	// f["test"] = test
	// i := f["test"]("!!!")
	// fmt.Println(string(i))
	if false {

		r := bufio.NewReader(os.Stdin)
		for {
			line, err := r.ReadString('\n')
			// buf =? buf[:]
			// if num > 3 {
			// 	os.Exit(0)
			// }
			//TODO: filterList
			if strings.Contains(line, "STARPORT]") || strings.Contains(line, "!") || strings.Contains(line, "lava_") || strings.Contains(line, "ERR_") {
				fmt.Println(string("!!! nice !!! ") + string(line))
			}
			// num += 1
			if err != nil {
				fmt.Print(string("XXX error XXX ") + string(err.Error()))
			}

			if err == io.EOF {
				break
			}
		}
	}

	// for {

	// 	n, err := r.Read(buf[:cap(buf)])
	// 	buf = buf[:n]

	// 	if n == 0 {

	// 		if err == nil {
	// 			continue
	// 		}

	// 		if err == io.EOF {
	// 			break
	// 		}

	// 		log.Fatal(err)
	// 	}

	// 	nChunks++
	// 	nBytes += int64(len(buf))

	// 	fmt.Println(string(buf))

	// 	if err != nil && err != io.EOF {
	// 		log.Fatal(err)
	// 	}
	// }

	// fmt.Println("Bytes:", nBytes, "Chunks:", nChunks)
}
