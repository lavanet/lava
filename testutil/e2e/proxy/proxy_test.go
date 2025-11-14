package mockproxy

import (
	"flag"
	"log"
	"os"
	"testing"
)

var (
	hostFlag = flag.String("host", "", "HOST (required) - Which host do you wish to proxy\nUsage Example:\n\t$ go run proxy.go http://google.com/")
	portFlag = flag.String("p", "1111", "PORT")
)

func TestStart(t *testing.T) {
	log.Println("Proxy Start args", os.Args[1:])
	main(hostFlag, portFlag)
}
