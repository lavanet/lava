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

func sleep(t int, failed *bool) {
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
