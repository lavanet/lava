package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

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

func sleep(t int) {
	if !*failed {
		for i := 1; i <= t; i++ {
			if *failed {
				break
			}
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
}

func getHomePath() string {
	userHome, err := os.UserHomeDir()
	if err != nil {
		panic(err.Error())
	}
	return userHome + "/"
}

func prepHomePath() (home string, isGithub bool) {
	isGithub = false
	homepath := os.Getenv("LAVA")
	if homepath == "" {
		homepath = getHomePath()
		if strings.Contains(homepath, "runner") { // on github
			homepath += "work/lava/lava"
			isGithub = true
		} else {
			homepath += "go/lava" //local
		}
	}
	homepath += "/"
	return homepath, isGithub
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
}

func killPid(pid int) bool {
	cmd := exec.Command("sh", "-c", "kill -9 "+fmt.Sprint(pid))
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Errorf(err.Error()))
	}
	fmt.Printf(" ::: XXXXX Killed Process %d %s\n", pid, stdoutStderr)
	return true
}
func isPidAlive(pid int) bool {
	cmd := exec.Command("sh", "-c", "ps -a | grep "+fmt.Sprint(pid))
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	res := string(stdoutStderr)
	return !strings.Contains(res, "defunct")
}

func strContains(line string, contains string) bool {
	return strings.Contains(line, contains)
}

func debugOn(state State) {
	*state.debug = true
}
func debugOff(state State) {
	*state.debug = false
}
func stdoutOn(state State) {
	*state.stdout = true
}
func stdoutOff(state State) {
	*state.stdout = false
}
func silent(state State) {
	fmt.Println("*********************************************")
	fmt.Println(" silencing " + state.id)
	fmt.Println("*********************************************")
	*state.debug = false
	*state.stdout = false
}
func stdout(state State) {
	*state.debug = true
	*state.stdout = true
}

func ExitLavaProcess() {
	cmd := exec.Command("sh", "-c", "killall lavad ; killall ignite ; killall starport ; killall main ; killall proxy")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Errorf(err.Error()).Error())
	}
	fmt.Printf(" ::: Exiting Lava Process ::: %s\n", stdoutStderr)
}
