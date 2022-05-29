//filter_pipe.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func mainB() {

	nBytes, nChunks := int64(0), int64(0)
	r := bufio.NewReader(os.Stdin)
	// buf := make([]byte, 0, 4*1024)
	num := 0
	filters := []string{"STARPORT", "!"}
	for {
		line, err := r.ReadString('\n')
		// buf =? buf[:]
		// if num > 3 {
		// 	os.Exit(0)
		// }
		found := false
		for _, filter := range filters {
			if strings.Contains(string(line), filter) {
				found = true
			}
		}
		if found {
			rline := strings.TrimSuffix(line, "\n")
			fmt.Println(string(" ... X") + string(rline) + "X")
		}
		num += 1
		if err != nil {
			fmt.Print(string("... error ... ") + string(err.Error()))
		}

		if err == io.EOF {
			break
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

	fmt.Println("Bytes:", nBytes, "Chunks:", nChunks)
}
