package mockproxy

import (
	"flag"
	"log"
	"os"
	"testing"
)

func TestStart(t *testing.T) {
	log.Println("Proxy Start", os.Args[len(os.Args)-1])

	host := &os.Args[len(os.Args)-1] //flag.String("host", "", "HOST (required) - Which host do you wish to proxy\nUsage Example:\n	$ go run proxy.go http://google.com/")
	port := flag.String("p", "1111", "PORT")
	main(host, port)
}
