//go:build !performance_tools

package performance

import "fmt"

func GetEntry(request []byte, block uint64, blockHash []byte, chainID string, apiInterface string) (exists bool, response []byte) {
	return false, nil
}

func SetEntry(request []byte, block uint64, blockHash []byte, chainID string, apiInterface string, response []byte) {
}

func Init() {
	fmt.Println("\nStub Cache\n")
}
