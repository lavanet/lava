package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var hostURL string = "mainnet.infura.io"

var mockFolder string = "testutil/e2e/proxy/mockMaps/"

var responsesChanged bool = false
var realCount int = 0
var cacheCount int = 0
var fakeCount int = 0

var fromCache bool = true
var fakeResponse bool = false
var saveCache bool = true

var saveJsonEvery int = 10 // in seconds
var epochTime int = 3      // in seconds
var epochCount int = 0     // starting epoch count
var proxies []proxyProcess = []proxyProcess{}

type proxyProcess struct {
	id        string
	port      string
	host      string
	mockfile  string
	mock      *mockMap
	handler   func(http.ResponseWriter, *http.Request)
	malicious bool
}

func getDomain(s string) (domain string) {
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return parts[len(parts)-2]
	}
	return s
}

func main() {
	startEpochUpdate()

	port := "2000"              // default
	host := "mainnet.infura.io" // default
	malicious := false          // default
	if len(os.Args) >= 2 {
		port = os.Args[1]
		if len(os.Args) >= 3 {
			host = os.Args[2]
			if len(os.Args) >= 4 {
				malicious = os.Args[3] == "1"
			}
		}
	}

	domain := getDomain(host)
	process := proxyProcess{
		id:        domain,
		port:      port,
		host:      host,
		mockfile:  mockFolder + domain + port + ".json",
		mock:      &mockMap{requests: map[string]string{}},
		malicious: malicious,
	}
	proxies = append(proxies, process)

	if !malicious {
		process.handler = process.LavaTestProxy
	} else {
		println()
		println("MMMMMMMMMMMMMMM MALICIOUS MMMMMMMMMMMMMMM PORT", port)
		println()
		// TODO: Make malicious proxy
		process.handler = process.LavaTestProxy
	}
	startProxyProcess(process)
}

func startProxyProcess(process proxyProcess) {
	process.mock.requests = jsonFileToMap(process.mockfile)
	if process.mock.requests == nil {
		process.mock.requests = map[string]string{}
	}
	if process.malicious {
		fakeResponse = true
	}
	fmt.Println("::::::::::::::::::::::::::::::::::::::::::::::: HOST ", process.host)
	fmt.Println("::::::::::::: Mock Proxy Started :::::::::::::: CACHE", fmt.Sprintf("%d", len(process.mock.requests)))
	fmt.Println("::::::::::::::::::::::::::::::::::::::::::::::: PORT ", process.port)
	println()

	http.HandleFunc("/", process.handler)
	err := http.ListenAndServe(":"+process.port, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func getMockBlockNumber() (block string) {
	return "0xe" + fmt.Sprintf("%d", epochCount) // return "0xe39ab8"
}

func startEpochUpdate() {
	go func() {
		count := 0
		for {
			wait := 1 * time.Second
			time.Sleep(wait)
			if count%epochTime == 0 {
				epochCount += 1
			}
			if saveCache && responsesChanged && count%saveJsonEvery == 0 {
				for _, process := range proxies {
					mapToJsonFile(*process.mock, process.mockfile)
				}
				responsesChanged = false
			}
			count += 1
		}
	}()
}

func (p proxyProcess) LavaTestProxy(rw http.ResponseWriter, req *http.Request) {
	// All incoming requests will be sent to this host
	host := hostURL
	mock := p.mock
	// Get request body
	rawBody := getDataFromIORead(&req.Body, true)
	println(p.port+" ::: INCOMING PROXY MSG :::", string(rawBody))

	// TODO: make generic
	// Check if asking for blockNumber
	if fakeResponse && strings.Contains(string(rawBody), "blockNumber") {
		println("!!!!!!!!!!!!!! block number")
		rw.WriteHeader(200)
		rw.Write([]byte(fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"%s\"}", getMockBlockNumber())))

	} else {
		// Return Cached data if found in history and fromCache is set on
		if val, ok := mock.requests[string(rawBody)]; ok && fromCache {
			println(p.port+" ::: Cached Response ::: ", string(val))
			cacheCount += 1

			// Change Response
			if fakeResponse {
				val = "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xe000000000000000000\"}"
				println(p.port+" ::: Fake Response ::: ", val)
				fakeCount += 1
			}
			rw.WriteHeader(200)
			rw.Write([]byte(val))

		} else {
			// Recreating Request
			proxyRequest, err := createProxyRequest(req, host)
			if err != nil {
				println(err.Error())
			}

			// Send Request to Host & Get Response
			proxyRes, err := sendRequest(proxyRequest)
			if err != nil {
				println(err.Error())
			}
			respBody := getDataFromIORead(&proxyRes.Body, true)
			mock.requests[string(rawBody)] = string(respBody)
			realCount += 1
			println(p.port+" ::: Real Response ::: ", string(respBody))

			// TODO: Check if response is good, if not - try again

			// Change Response
			if fakeResponse {
				respBody = []byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xe000000000000000000\"}")
				println(p.port+" ::: Fake Response ::: ", string(respBody))
				fakeCount += 1
			}
			responsesChanged = true

			//Return Response
			returnResponse(rw, proxyRes.StatusCode, respBody)
		}
	}
	println("_________________________________", realCount, "/", cacheCount, "\n")
}
