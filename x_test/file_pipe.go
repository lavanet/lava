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

func readFile(path string, done *bool, doneF func(string) bool) {
	file, fileError := os.Open(path)
	if fileError != nil {
		panic(fileError.Error())
	}
	defer file.Close()
	r := bufio.NewReader(file)
	for {
		// line, err := r.ReadString('\n')
		data, _, err := r.ReadLine()
		// num += 1
		if err != nil {
			// fmt.Println(string("XXX error XXX ") + string(err.Error()))
		} else {
			line := string(data)
			if processLog(line, path, doneF) {
				*done = true
			}
		}

		if err == io.EOF {
			// fmt.Printf("EOF.")
			// break
		}
	}
}

func processLog(line string, parent string, doneF func(string) bool) bool {
	if strings.Contains(line, "STARPORT]") || strings.Contains(line, "!") || strings.Contains(line, "lava_") || strings.Contains(line, "ERR_") {
		fmt.Println(parent + " ::: " + string("!!! lava !!! ") + string(line))
	} else {
		// fmt.Println(parent + " ::: " + string("!!! nice !!! ") + string(line))
	}
	return doneF(line)
}
func ExampleCmd_CombinedOutput() {
	cmd := exec.Command("sh", "-c", "echo stdout; echo 1>&2 stderr")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", stdoutStderr)
}

type State struct {
	finished    bool `json:"finished"`
	nodeUp      bool `json:"finished"`
	providersUp bool `json:"strict"`
}

func logProcess(id string, home string, cmd string, done *bool, doneF func(string) bool) *bool {
	os.Chdir(home)
	logFile := id + ".log"
	logPath := resetLog(home, logFile, "x_test/tests/integration/")
	// logPath := home + logFile
	end := " 2>&1"
	println("home", home)
	full := "cd " + home + " && " + cmd + " >> " + logPath + end
	println("full", full)
	fullCMD := exec.Command("sh", "-c", full)
	// fullCMD := exec.Command(cmd + end + " >> " + logPath + ")")
	go fullCMD.Start()
	go readFile(logPath, done, doneF)
	return done
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

func advanceFlow_init(line string) bool {
	contains := "lava_client_stake"
	return advanceFlow(line, contains)
}

func advanceFlow_client(line string) bool {
	contains := "lava_relay_payment"
	return advanceFlow(line, contains)
}
func advanceFlow_node(line string) bool {
	contains := "🌍 Token faucet: http"
	return advanceFlow(line, contains)
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
func await(done *bool, msg string, after string) {
	for !*done {
		time.Sleep(1 * time.Second)
		println(" ::: " + msg)
	}
	println("*********************************************")
	println(fmt.Sprintf("***************  %s   *********************", after))
	println("*********************************************")
}
func main() {

	homepath := getHomePath() + "go/lava/"
	done := false
	go logProcess("starport", homepath, "killall starport; starport chain serve -v -r ", &done, advanceFlow_node)
	// go logProcess("starport", homepath, "lavad start", &done, advanceFlow_node)
	// go logProcess("starport", homepath, "starport chain serve -v -r |& grep -e lava_ -e ERR_ -e STARPORT] -e !", &done, true)
	// time.Sleep(30 * time.Second)
	await(&done, "awating for node to proceed...", "init")
	time.Sleep(1 * time.Second)

	done2 := false
	go logProcess("init", homepath, "./init_chain_commands.sh", &done2, advanceFlow_init)
	await(&done2, "awating stake to proceed...", "client")

	done3 := false
	go logProcess("client", homepath, "go run /home/magic/go/lava/relayer/cmd/relayer/main.go test_client ETH1 jsonrpc --from user1", &done3, advanceFlow_client)
	await(&done3, "awating payment to proceed...", "DONE!!!")

}
