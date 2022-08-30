package common

import (
	"fmt"
	"runtime"
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
	lockCount   int
}

func (dm *LavaMutex) getLineAndFile() string {
	_, file, line, _ := runtime.Caller(2)
	return fmt.Sprintf("%s:%d", file, line)
	// var buf [512]byte

	// runtime.Stack(buf[:], true)
	// temp := strings.Split(string(buf[:]), "\n")
	// filepath := ""
	// if len(temp) < 6 {
	// 	filepath = temp[len(temp)]
	// } else {
	// 	filepath = temp[6]
	// }
	// filepath = strings.Replace(filepath, "\t", "", -1)
	// split := strings.Split(filepath, ":")
	// path, lineNumStr := split[0], split[1]
	// lineNumStr = strings.Split(lineNumStr, " ")[0]
	// lineNum, err := strconv.Atoi(lineNumStr)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return ""
	// }
	// file, err := os.Open(path)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer file.Close()

	// scanner := bufio.NewScanner(file)
	// i := 1
	// for scanner.Scan() {
	// 	if i == lineNum {
	// 		return fmt.Sprintf("%s:%s: %s", path, lineNumStr, strings.TrimSpace(scanner.Text()))
	// 	}

	// 	i = i + 1
	// }
	// if err := scanner.Err(); err != nil {
	// 	log.Fatal(err)
	// }
	// return ""
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
				fmt.Printf("WARNING: Mutex is Locked for more than %d seconds \n %s \n", TIMEOUT, dm.lineAndFile)
				return
			}
		}
	}()

}

func (dm *LavaMutex) Lock() {
	tempLineAndFile := dm.getLineAndFile()
	dm.lockCount = dm.lockCount + 1
	fmt.Printf("Lock: %s, count %d ... ", tempLineAndFile, dm.lockCount)
	dm.mu.Lock()
	fmt.Printf("locked \n")
	dm.lineAndFile = tempLineAndFile
	dm.SecondsLeft = TIMEOUT
	dm.waitForTimeout()
}

func (dm *LavaMutex) TryLock() (isLocked bool) {
	tempLineAndFile := dm.getLineAndFile()

	isLocked = dm.mu.TryLock()
	if isLocked {
		dm.lockCount = dm.lockCount + 1
		fmt.Println("TryLock Locked: ", tempLineAndFile)
		dm.lineAndFile = tempLineAndFile
		dm.SecondsLeft = TIMEOUT
		dm.waitForTimeout()
	}
	return isLocked
}

func (dm *LavaMutex) Unlock() {
	fmt.Println("Unlock: ", dm.getLineAndFile())
	dm.lockCount = dm.lockCount - 1
	dm.quit <- true
	dm.mu.Unlock()
}

// func main() {
// 	x := LavaMutex{}
// 	x.Lock()

// 	time.Sleep(6 * time.Second)

// 	x.Unlock()

// 	return
// }
