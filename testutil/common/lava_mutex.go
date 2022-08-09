package common

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const TIMEOUT = 5

type LavaMutex struct {
	mu          sync.Mutex
	quit        chan bool
	SecondsLeft int
	lineAndFile string
	hasQuit     bool
}

func (dm *LavaMutex) getLineAndFile() string {
	var buf [512]byte
	runtime.Stack(buf[:], true)
	temp := strings.Split(string(buf[:]), "\n")
	filepath := temp[6]
	filepath = strings.Replace(filepath, "\t", "", -1)
	split := strings.Split(filepath, ":")
	path, lineNumStr := split[0], split[1]
	lineNumStr = strings.Split(lineNumStr, " ")[0]
	lineNum, err := strconv.Atoi(lineNumStr)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	i := 1
	for scanner.Scan() {
		if i == lineNum {
			return fmt.Sprintf("%s:%s: %s", path, lineNumStr, strings.TrimSpace(scanner.Text()))
		}

		i = i + 1
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return ""
}

func (dm *LavaMutex) waitForTimeout() {
	dm.quit = make(chan bool)
	ticker := time.NewTicker(TIMEOUT * time.Second)
	go func() {
		for {
			select {
			case <-dm.quit:
				ticker.Stop()
				return
			case <-ticker.C:
				ticker.Stop()
				fmt.Printf("WARNING: Mutex is Locked for more than %d seconds \n", TIMEOUT)
				return
			}
		}
	}()

}

func (dm *LavaMutex) Lock() {
	tempLineAndFile := dm.getLineAndFile()
	fmt.Println("Lock: ", tempLineAndFile)
	dm.mu.Lock()
	dm.lineAndFile = tempLineAndFile
	dm.SecondsLeft = TIMEOUT
	dm.waitForTimeout()
}

func (dm *LavaMutex) Unlock() {
	fmt.Println("Unlock: ", dm.getLineAndFile())
	dm.mu.Unlock()
	dm.quit <- true
}

// func main() {
// 	x := LavaMutex{}
// 	x.Lock()

// 	time.Sleep(6 * time.Second)

// 	x.Unlock()

// 	return
// }
