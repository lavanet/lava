package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

var hostURL string = "mainnet.infura.io"
var mock mockMap = mockMap{map[string]string{}}
var responsesChanged bool = false
var realCount int = 0
var cacheCount int = 0
var fakeCount int = 0

// var handler func(http.ResponseWriter, *http.Request) = basicProxy
var handler func(http.ResponseWriter, *http.Request) = lavaTestProxy

var fromCache bool = true
var fakeResponse bool = false
var saveCache bool = true

var saveJsonEvery int = 10 // in seconds
var epochTime int = 3      // in seconds
var epochCount int = 0     // starting epoch count

func main() {
	go func() {
		count := 0
		for {
			wait := 1 * time.Second
			time.Sleep(wait)
			if count%epochTime == 0 {
				epochCount += 1
			}
			if saveCache && responsesChanged && count%saveJsonEvery == 0 {
				mapToJsonFile(mock, "./"+mockFile)
				responsesChanged = false
			}
			count += 1
		}
	}()
	mock.requests = jsonFileToMap(mockFile)
	if mock.requests == nil {
		mock.requests = map[string]string{}
	}
	fmt.Println(":::::::::::::::::::::::::::::::::::::::::::::::")
	fmt.Println("::::::::::::: Mock Proxy Started ::::::::::::::")
	fmt.Println(":::::::::::::::::::::::::::::::::::::::::::::::")
	println()
	http.HandleFunc("/", handler)
	http.ListenAndServe(":2000", nil)
}

func getMockBlockNumber() (block string) {
	return "0xe" + fmt.Sprintf("%d", epochCount) // return "0xe39ab8"
}

func lavaTestProxy(rw http.ResponseWriter, req *http.Request) {

	// All incoming requests will be sent to this host
	host := hostURL

	// Get request body
	rawBody := getDataFromIORead(&req.Body, true)
	println(" ::: INCOMING PROXY MSG :::", string(rawBody))

	// TODO: make generic
	// Check if asking for blockNumber
	if fakeResponse && strings.Contains(string(rawBody), "blockNumber") {
		println("!!!!!!!!!!!!!! block number")
		rw.WriteHeader(200)
		rw.Write([]byte(fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"%s\"}", getMockBlockNumber())))

	} else {

		// Return Cached data if found in history and fromCache is set on
		if val, ok := mock.requests[string(rawBody)]; ok && fromCache {
			// println(" ::::::::::: FROM CACHE ::::::::::::: ")
			println(" ::: Cached Response ::: ", string(val))
			cacheCount += 1
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
			println(" ::: Real Response ::: ", string(respBody))

			// TODO: Check if response is good, if not - try again

			// Change Response
			if fakeResponse {
				respBody = []byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xe000000000000000000\"}")
				println(" ::: Fake Response ::: ", string(respBody))
				fakeCount += 1
			}
			responsesChanged = true

			//Return Response
			returnResponse(rw, proxyRes.StatusCode, respBody)
		}
	}
	println("_________________________________", realCount, "/", cacheCount, "\n")
}
