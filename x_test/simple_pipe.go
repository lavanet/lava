//simple_pipe.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	r := bufio.NewReader(os.Stdin)
	for {
		line, err := r.ReadString('\n')
		if strings.Contains(line, "STARPORT]") || strings.Contains(line, "!") || strings.Contains(line, "lava_") || strings.Contains(line, "ERR_") {
			fmt.Println(string("!!! lava !!! ") + string(line))
		} else {
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
