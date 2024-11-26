package mockproxy

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcInterfaceMessages"
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
)

// var mockFolder string = "testutil/e2e/proxy/mockMaps/"
var mockFolder string = "./mockMaps/"

var (
	responsesChanged bool = false
	realCount        int  = 0
	cacheCount       int  = 0
	fakeCount        int  = 0
)

var fakeResponse bool = false

var (
	saveJsonEvery int            = 10 // in seconds
	epochTime     int            = 3  // in seconds
	epochCount    int            = 0  // starting epoch count
	proxies       []proxyProcess = []proxyProcess{}
)

type proxyProcess struct {
	id        string
	port      string
	host      string
	mockfile  string
	mock      *mockMap
	handler   func(http.ResponseWriter, *http.Request)
	malicious bool
	cache     bool
	strict    bool
	noSave    bool
}

type jsonStruct struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      int           `json:"id,omitempty"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params,omitempty"`
}

func getDomain(s string) (domain string) {
	parts := strings.Split(s, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return s
}

var current *proxyProcess

func main(host, port *string) {
	// CLI ARGS
	fmt.Println("Proxy Main")
	// host := flag.String("host", "", "HOST (required) - Which host do you wish to proxy\nUsage Example:\n	$ go run proxy.go http://google.com/")
	// port := flag.String("p", "1111", "PORT")
	id := flag.String("id", "", "ID (optional) - will set the default host id instead of the full domain name"+
		"\nUsage Example:\n	$ go run proxy.go randomnumberapi.com -id random -cache")
	cache := flag.Bool("cache", true, "CACHE (optional) - This will make proxy return from cache if possible "+
		"(from default [host].json unless -alt was set)\nUsage Example:\n	$ go run proxy.go http://google.com/ -cache")
	alt := flag.String("alt", "", "ALT (optional) [JSONFILE] - This will make proxy return from alternative cache file if possible"+
		"\nUsage Example:\n	$ go run proxy.go http://google.com/ -cache -alt ./mockMaps/google_alt.json		# respond from google_alt.json")
	strict := flag.Bool("strict", true, "STRICT (optional) - This will make proxy return ONLY from cache, no external calls")
	help := flag.Bool("h", false, "Shows this help message")
	noSave := flag.Bool("no-save", true, "NO-SAVE (optional) will not store any data from proxy")

	flag.Parse()
	if *help || (*host == "" && flag.NArg() == 0) {
		fmt.Println("go run proxy.go [host] -p [port] OPTIONAL -cache -alt [JSONFILE] -strict")
		fmt.Println("	Usage Example:")
		fmt.Println("	$ go run proxy.go -host google.com/ -p 1111 -cache ")
		flag.Usage()
	} else if *host == "" {
		if len(os.Args) > 0 {
			if os.Args[1] != "-host" {
				*host = os.Args[1]
				flag.CommandLine.Parse(append([]string{"-host"}, os.Args[1:]...))
			} else {
				*host = os.Args[1]
			}
		}
	}
	println()

	domain := getDomain(*host)
	println("domain:" + domain)
	if *id != "" {
		domain = *id
	} else {
		*id = domain
	}

	mockfile := mockFolder + domain + ".json"
	if *alt != "" {
		mockfile = mockFolder + *alt
	}

	if *host == "" {
		println("\n [host] is required. Exiting")
		os.Exit(1)
	}
	malicious := false // default

	startEpochUpdate(*noSave)

	process := proxyProcess{
		id:        domain,
		port:      *port,
		host:      *host,
		mockfile:  mockfile,
		mock:      &mockMap{requests: map[string]string{}},
		malicious: false,
		cache:     *cache,
		strict:    *strict,
		noSave:    *noSave,
	}
	current = &process
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

// Define the upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func startProxyProcess(process proxyProcess) {
	process.mock.requests = jsonFileToMap(process.mockfile)
	if process.mock.requests == nil {
		process.mock.requests = map[string]string{}
	}
	if process.malicious {
		fakeResponse = true
	}
	fmt.Println(":::::::::::::::::::::::::::::::::::::::::::::::") // HOST ", process.host)
	fmt.Println("::::::::::::: Mock Proxy Started ::::::::::::::") // CACHE", fmt.Sprintf("%d", len(process.mock.requests)))
	fmt.Println(":::::::::::::::::::::::::::::::::::::::::::::::") // PORT ", process.port)
	println()
	fmt.Print(fmt.Sprintf(" ::: Proxy ID 		::: %s", process.id) + "\n")
	fmt.Print(fmt.Sprintf(" ::: Proxy Host 	::: %s", process.host) + "\n")
	fmt.Print(fmt.Sprintf(" ::: Return Cache 	::: %t", process.cache) + "\n")
	fmt.Print(fmt.Sprintf(" ::: Strict Mode 	::: %t", process.strict) + "\n")
	fmt.Print(fmt.Sprintf(" ::: Saving	 	::: %t", !process.noSave) + "\n")
	if !process.noSave || process.cache {
		fmt.Print(fmt.Sprintf(" ::: Cache File 	::: %s", process.mockfile) + "\n")
		fmt.Print(fmt.Sprintf(" ::: Loaded Responses 	::: %d", len(process.mock.requests)) + "\n")
	}
	println()
	fmt.Print(fmt.Sprintf(" ::: Proxy Started! 	::: ID: %s", process.id) + "\n")
	fmt.Print(fmt.Sprintf(" ::: Listening On 	::: %s", "http://0.0.0.0:"+process.port+"/") + "\n")

	// Define the WebSocket handler
	wsHandler := func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		defer conn.Close()

		for {
			// Read message from browser
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Read error:", err)
				break
			}
			// Print the message to the console
			log.Printf("WS Received: %s\n", msg)

			var respmsg rpcclient.JsonrpcMessage
			err = json.Unmarshal(msg, &respmsg)
			if err != nil {
				println(err.Error())
				continue
			}

			replyMessage, err := rpcInterfaceMessages.ConvertJsonRPCMsg(&respmsg)
			if err != nil {
				println(err.Error())
				continue
			}

			jStruct := &jsonStruct{}
			err = json.Unmarshal(msg, jStruct)
			if err != nil {
				println(err.Error())
				continue
			}
			jStruct.ID = 0
			rawBodySNoID, _ := json.Marshal(jStruct)

			if val, ok := process.mock.requests[string(rawBodySNoID)]; ok && process.cache {
				orderedJSON := idInsertedResponse(val, replyMessage)
				println(dotsStr+process.port+dotsStr+process.id+" ::: Cached Response ::: ", orderedJSON)
				cacheCount += 1

				// Change Response
				if fakeResponse {
					val = fakeResult(val, "0xe000000000000000000")
					// val = "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xe000000000000000000\"}"
					println(process.port+" ::: Fake Response ::: ", val)
					fakeCount += 1
				}
				time.Sleep(500 * time.Millisecond)
				conn.WriteMessage(msgType, []byte(orderedJSON))
			} else if err = conn.WriteMessage(msgType, msg); err != nil {
				// Write message back to browser
				log.Println("Write error:", err)
				break
			}
		}
	}

	http.HandleFunc("/", process.handler)
	http.HandleFunc("/ws", wsHandler)
	err := http.ListenAndServe(":"+process.port, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func getMockBlockNumber() (block string) {
	return "0xe" + fmt.Sprintf("%d", epochCount) // return "0xe39ab8"
}

func startEpochUpdate(noSave bool) {
	go func() {
		count := 0
		for {
			wait := 1 * time.Second
			time.Sleep(wait)
			if count%epochTime == 0 {
				epochCount += 1
			}
			if !noSave && responsesChanged && count%saveJsonEvery == 0 {
				for _, process := range proxies {
					mapToJsonFile(*process.mock, process.mockfile)
				}
				responsesChanged = false
			}
			count += 1
		}
	}()
}

func fakeResult(val, fake string) string {
	parts := strings.Split(val, ",")
	found := -1
	for i, part := range parts {
		if strings.Contains(part, "result") {
			found = i
		}
	}
	if found != -1 {
		parts[found] = fmt.Sprintf("\"result\":\"%s\"}", fake)
	}
	return strings.Join(parts, ",")
}

func idInsertedResponse(val string, replyMessage *rpcInterfaceMessages.JsonrpcMessage) string {
	// Extract ID from raw message
	respId, idErr := rpcInterfaceMessages.IdFromRawMessage(replyMessage.ID)
	if idErr != nil {
		println("Failed extract ID from raw message")
	}
	// Parse the JSON string into a map
	var responseMap map[string]interface{}
	if err := json.Unmarshal([]byte(val), &responseMap); err != nil {
		println("Failed to unmarshal JSON:", err.Error())
		return ""
	}
	// Extract the "result" field
	result, resultExists := responseMap["result"]
	if !resultExists {
		println("Result field is missing")
		return ""
	}
	// Convert the "result" field back to a JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		println("Failed to marshal result JSON:", err.Error())
		return ""
	}
	// Manually construct the JSON string with "id" field updated and in desired order
	return fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":%s}", respId, string(resultJSON))
}

const dotsStr = " ::: "

func (p proxyProcess) LavaTestProxy(responseWriter http.ResponseWriter, request *http.Request) {
	host := p.host
	mock := p.mock

	// Get request body
	rawBody := getDataFromIORead(&request.Body, true)
	// TODO: set all ids to 1
	rawBodyS := string(rawBody)
	// sep := "id\":"
	// first := strings.Split(rawBodyS, sep)[0]
	// second := strings.Split(rawBodyS, sep)[1]
	// second = strings.Join(strings.Split(second, ",")[1:], ",")
	// rawBodyS = first + sep + "1," + second

	println()
	println(dotsStr+p.port+dotsStr+p.id+" ::: INCOMING PROXY MSG :::", rawBodyS)

	var respmsg rpcclient.JsonrpcMessage
	if err := json.NewDecoder(request.Body).Decode(&respmsg); err != nil {
		println(err.Error())
	}
	replyMessage, err := rpcInterfaceMessages.ConvertJsonRPCMsg(&respmsg)
	if err != nil {
		println(err.Error())
	}
	respId, idErr := rpcInterfaceMessages.IdFromRawMessage(replyMessage.ID)
	if idErr != nil {
		println(idErr.Error())
	}

	if fakeResponse && strings.Contains(rawBodyS, "blockNumber") {
		println("!!!!!!!!!!!!!! block number")
		returnResponse(responseWriter, http.StatusOK, []byte(fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":\"%s\"}", respId, getMockBlockNumber())))
	} else {
		// Return Cached data if found in history and fromCache is set on
		jStruct := &jsonStruct{}
		json.Unmarshal([]byte(rawBodyS), jStruct)
		jStruct.ID = 0
		rawBodySNoID, _ := json.Marshal(jStruct)
		if val, ok := mock.requests[string(rawBodySNoID)]; ok && p.cache {
			orderedJSON := idInsertedResponse(val, replyMessage)
			println(dotsStr+p.port+dotsStr+p.id+" ::: Cached Response ::: ", orderedJSON)
			cacheCount += 1

			// Change Response
			if fakeResponse {
				val = fakeResult(val, "0xe000000000000000000")
				// val = "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xe000000000000000000\"}"
				println(p.port+" ::: Fake Response ::: ", val)
				fakeCount += 1
			}
			time.Sleep(500 * time.Millisecond)
			returnResponse(responseWriter, http.StatusOK, []byte(orderedJSON))
		} else {
			// Recreating Request
			proxyRequest, err := createProxyRequest(request, host, rawBodyS)
			if err != nil {
				println(err.Error())
			} else {
				// Send Request to Host & Get Response
				proxyRes, err := sendRequest(proxyRequest)
				// respBody := []byte("error")
				var respBody []byte
				respBodyStr := "xxxxxx"
				status := 400
				if err != nil {
					println(err.Error())
					respBody = []byte(err.Error())
				} else {
					status = proxyRes.StatusCode
					respBody = getDataFromIORead(&proxyRes.Body, true)
					respBodyStr = string(respBody)
					mock.requests[rawBodyS] = respBodyStr
					realCount += 1
					println(dotsStr+p.port+dotsStr+p.id+" ::: Real Response ::: ", respBodyStr)
				}

				// Check if response is not good, if not - try again
				if false && (strings.Contains(string(respBody), "error") || strings.Contains(string(respBody), "Error")) {
					println("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX Got error in response - retrying request")

					// Recreating Request
					proxyRequest, err = createProxyRequest(request, host, rawBodyS)
					if err != nil {
						println(err.Error())
						respBody = []byte(err.Error())
					} else {
						// Send Request to Host & Get Response
						proxyRes, err = sendRequest(proxyRequest)
						if err != nil {
							println(err.Error())
							respBody = []byte("error: " + err.Error())
						} else {
							respBody = getDataFromIORead(&proxyRes.Body, true)
							mock.requests[rawBodyS] = string(respBody)
							status = proxyRes.StatusCode
						}
						realCount += 1
						println(dotsStr+p.port+dotsStr+p.id+" ::: Real Response ::: ", string(respBody))

						// TODO: Check if response is good, if not - try again
						if strings.Contains(string(respBody), "error") || strings.Contains(string(respBody), "Error") {
							println("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX Got another error in response ")
							println()
						} else {
							println("YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY SUCCESS - no error in response ")
							println()
						}
					}
				}

				// Change Response
				if fakeResponse {
					// respBody = []byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xe000000000000000000\"}")
					respBody = []byte(fakeResult(respBodyStr, "0xe000000000000000000"))
					println(dotsStr+p.port+dotsStr+p.id+" ::: Fake Response ::: ", string(respBody))
					fakeCount += 1
				}
				responsesChanged = true

				// Return Response
				if respBody == nil {
					respBody = []byte("error")
				}
				// time.Sleep(500 * time.Millisecond)
				returnResponse(responseWriter, status, respBody)
			}
		}
	}
	if realCount > 0 || cacheCount > 0 {
		id := ""
		if current != nil {
			id = current.id + ":" + current.port
		}
		fmt.Println("_________________________________", realCount, "/", cacheCount, ": proxy sent (new/from cache)", id, "")
	}
}
