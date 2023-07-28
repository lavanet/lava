package main

import (
	"fmt"

	lavavisor "github.com/lavanet/lava/ecosystem/lavavisor/cmd"
)

func main() {
	fmt.Println("Starting LavaVisor...")
	lavavisor.Execute()
}
