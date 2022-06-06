package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

var responses map[string]string = map[string]string{}

type mockMap struct {
	mock map[string]string `json:"mock"`
}

var blockCount int
var mockFile string

func main() {
	blockCount = 0
	mockFile = "testutil/e2e/proxy/mock.json"
	go func() {
		for true {
			time.Sleep(1 * time.Second)
			blockCount += 1
		}
	}()
	responses = jsonFileToMap(mockFile)
	if responses == nil {
		responses = map[string]string{}
	}
	fmt.Printf(":::::::::::::::::::::::::::\n")
	// responses := make(chan map[string][]byte)
	// go func() {
	// 	// run lava testing
	// 	// Test finished on time !
	// 	responses <- map[string][]byte{"ok": []byte("nice")}
	// }()
	// res := <-responses
	// fmt.Printf(":::::::::::::::::::::::::::\n", res)
	// fmt.Printf(":::::::::::::::::::::::::::\n")
	// http.HandleFunc("/", handler5)
	http.HandleFunc("/", handler5)
	http.ListenAndServe(":2000", nil)
}

func mapToJsonFile(m map[string]string, outfile string) error {
	// map to json
	// json to file
	jsonData, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
		return err
	}
	jsonFile, err := os.Create(outfile)

	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	jsonFile.Write(jsonData)
	jsonFile.Close()
	fmt.Println("JSON data written to ", jsonFile.Name())

	return nil
}
func jsonFileToMap(jsonfile string) (m map[string]string) {
	// open file
	// unmarshal to mockMap
	m = map[string]string{}
	jsonFile, err := os.Open(jsonfile)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(" ::: XXX ::: Could not open "+jsonfile+" ::: ", err)
		return
	}
	fmt.Println(" ::: Successfully Opened " + jsonfile)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	// read our opened jsonFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we initialize our Users array
	var mock mockMap
	for key, body := range mock.mock {
		m[key] = body
	}

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &mock)
	return
}

// type Payload struct {
// 	Jsonrpc string        `json:"jsonrpc"`
// 	Method  string        `json:"method"`
// 	Params  []interface{} `json:"params"`
// 	ID      int           `json:"id"`
// }

func getMockBlockNumber() (block string) {
	return "0xe39ab8"
}

func handler5(rw http.ResponseWriter, req *http.Request) {
	url := req.URL
	url.Host = fmt.Sprintf("mainnet.infura.io")
	// responses = jsonFileToMap("mock.json")

	// r := io.Reader(req.Body)
	// var buf bytes.Buffer
	// tee := io.TeeReader(r, &buf)
	rawBody, _ := ioutil.ReadAll(req.Body)
	println("RRRRRRRRRRRRRR", string(rawBody))
	if strings.Contains(string(rawBody), "blockNumber") {
		print("!!!!!!!!!!!!!! block number")
		rw.WriteHeader(200)
		rw.Write([]byte(fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"%s\"}", getMockBlockNumber())))

	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(rawBody))

	if val, ok := responses[string(rawBody)]; ok {
		println(" ::::::::::: FROM CACHE ::::::::::::: ")
		rw.WriteHeader(200)
		rw.Write([]byte(val))
	} else {
		println(" ::::::::::: FROM REAL ::::::::::::: ")
	}

	proxyReq, err := http.NewRequest(req.Method, url.String(), req.Body)
	// proxyReq, err := http.NewRequest(req.Method, url.String(), tee)

	if err != nil {
		println("XXXXXX ERR 1", err.Error(), url.Host)
	}

	proxyReq.Header.Set("Host", req.Host)
	proxyReq.Header.Set("X-Forwarded-For", req.RemoteAddr)
	// proxyReq.RequestURI = "v3/3755a1321ab24f938589412403c46455"
	proxyReq.URL.Scheme = "https"
	for header, values := range req.Header {
		for _, value := range values {
			// println("!!!!!!!!!!!!!! ", header, value)
			proxyReq.Header.Add(header, value)
		}
	}

	client := &http.Client{}
	proxyRes, err := client.Do(proxyReq)
	if err != nil {
		println("XXXXXX ERR 2", err.Error())
	}
	// reqBod, err := ioutil.ReadAll(req.Body)
	// println("RRRRRRRR", string(reqBod))

	respBody, _ := ioutil.ReadAll(proxyRes.Body)

	// if req != nil && req.Body != nil {
	// 	reqBody, _ := ioutil.ReadAll(req.Body)
	// 	println("RRRRRRRRRR ::: [", string(reqBody), "]")
	// }

	println("Real Response ::: ", string(respBody))
	// respBody = []byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xe000000000000000000\"}")
	// println("Fake Response ::: ", string(respBody))
	// Here:

	responses[string(rawBody)] = string(respBody)
	// mapToJsonFile(responses, "./mock.json")
	mapToJsonFile(responses, "./"+mockFile)
	rw.WriteHeader(proxyRes.StatusCode)
	rw.Write(respBody)

}

func handler4(rw http.ResponseWriter, req *http.Request) {
	url := req.URL
	url.Host = fmt.Sprintf("mainnet.infura.io")

	// r := io.Reader(req.Body)
	// var buf bytes.Buffer
	// tee := io.TeeReader(r, &buf)
	rawBody, _ := ioutil.ReadAll(req.Body)
	println("RRRRRRRRRRRRRR", string(rawBody))
	req.Body = ioutil.NopCloser(bytes.NewBuffer(rawBody))

	proxyReq, err := http.NewRequest(req.Method, url.String(), req.Body)
	// proxyReq, err := http.NewRequest(req.Method, url.String(), tee)

	if err != nil {
		println("XXXXXX ERR 1", err.Error(), url.Host)
	}

	proxyReq.Header.Set("Host", req.Host)
	proxyReq.Header.Set("X-Forwarded-For", req.RemoteAddr)
	// proxyReq.RequestURI = "v3/3755a1321ab24f938589412403c46455"
	proxyReq.URL.Scheme = "https"
	for header, values := range req.Header {
		for _, value := range values {
			// println("!!!!!!!!!!!!!! ", header, value)
			proxyReq.Header.Add(header, value)
		}
	}

	client := &http.Client{}
	proxyRes, err := client.Do(proxyReq)
	if err != nil {
		println("XXXXXX ERR 2", err.Error())
	}
	// reqBod, err := ioutil.ReadAll(req.Body)
	// println("RRRRRRRR", string(reqBod))

	respBody, _ := ioutil.ReadAll(proxyRes.Body)

	// if req != nil && req.Body != nil {
	// 	reqBody, _ := ioutil.ReadAll(req.Body)
	// 	println("RRRRRRRRRR ::: [", string(reqBody), "]")
	// }

	println("Real Response ::: ", string(respBody))
	// respBody = []byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xe000000000000000000\"}")
	// println("Fake Response ::: ", string(respBody))
	// Here:
	rw.WriteHeader(proxyRes.StatusCode)
	rw.Write(respBody)

}

func handler3(rw http.ResponseWriter, req *http.Request) {
	// res := <-responses
	fmt.Printf(":H:::::::::::::::::::::::::\n")
	// fmt.Printf(":::::::::::::::::::::::::::\n", responses)

	url := req.URL
	url.Host = fmt.Sprintf("mainnet.infura.io")

	proxyReq, err := http.NewRequest(req.Method, url.String(), req.Body)
	if err != nil {
		println("XXXXXX ERR 1", err.Error(), url.Host)
	}

	reqBody, _ := ioutil.ReadAll(proxyReq.Body)
	// current := <-responses
	if val, ok := responses[string(reqBody)]; ok {
		println("!!!!!! FROM CACHE !!!!!!!! ", string(reqBody))
		rw.WriteHeader(200)
		rw.Write([]byte(val))

	} else {
		println("!!!!!! FROM SERVER !!!!!!!! ", string(reqBody))
		proxyReq.Header.Set("Host", req.Host)
		proxyReq.Header.Set("X-Forwarded-For", req.RemoteAddr)
		// proxyReq.RequestURI = "v3/3755a1321ab24f938589412403c46455"
		proxyReq.URL.Scheme = "https"
		for header, values := range req.Header {
			for _, value := range values {
				// println("!!!!!!!!!!!!!! ", header, value)
				proxyReq.Header.Add(header, value)
			}
		}

		client := &http.Client{}
		proxyRes, err := client.Do(proxyReq)
		if err != nil {
			println("XXXXXX ERR 2", err.Error())
		}

		println("RRRRRRRRRR ::: [", string(reqBody), "]")
		respBody, _ := ioutil.ReadAll(proxyRes.Body)
		// current := <-responses
		// responses <- current
		println("Real Response ::: ", string(respBody))
		rw.WriteHeader(proxyRes.StatusCode)
		rw.Write(respBody)
		responses[string(reqBody)] = string(respBody)
		mapToJsonFile(responses, "./mock.json")
		// respBody = []byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xe000000000000000000\"}")
		// println("Fake Response ::: ", string(respBody))
		// Here:
	}

}

func handler2(w http.ResponseWriter, req *http.Request) {
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go

	// curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' 'https://mainnet.infura.io/v3/3755a1321ab24f938589412403c46455'
	fmt.Printf("req.Body: %v\n", req)
	type Payload struct {
		Jsonrpc string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      int           `json:"id"`
	}

	data := Payload{
		Jsonrpc: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      1,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		// handle err
	}
	body := bytes.NewReader(payloadBytes)

	res, err := http.NewRequest("POST", "https://mainnet.infura.io/v3/3755a1321ab24f938589412403c46455", body)
	if err != nil {
		// handle err
	}
	res.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(res)

	if err != nil {
		println("XXXXXXXXX", err.Error())
		// return nil, err
	}
	println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", resp.Body)

	defer resp.Body.Close()

}

func handler(w http.ResponseWriter, req *http.Request) {
	fmt.Println(":::::::::::::::::::::::::::")
	// we need to buffer the body if we want to read it here and send it
	// in the request.
	r := req
	r.URL.Host = "https://mainnet.infura.io/v3/3755a1321ab24f938589412403c46455/eth/ws/"
	r.RequestURI = ""
	client := &http.Client{}

	// delete(r.Header, "Accept-Encoding")
	// delete(r.Header, "Content-Length")
	resp, err := client.Do(r.WithContext(context.Background()))
	if err != nil {
		println("XXXXXXXXX", err.Error())
		// return nil, err
	}
	// return resp, nil
	fmt.Printf("resp.Body: %v\n", resp.Body)
	println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", resp.Body)

	// body, err := ioutil.ReadAll(req.Body)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	// // you can reassign the body if you need to parse it as multipart
	// req.Body = ioutil.NopCloser(bytes.NewReader(body))

	// // create a new url from the raw RequestURI sent by the client
	// // url := fmt.Sprintf("%s://%s%s", proxyScheme, proxyHost, req.RequestURI)
	// url := "https://mainnet.infura.io/v3/3755a1321ab24f938589412403c46455"

	// proxyReq, err := http.NewRequest(req.Method, url, bytes.NewReader(body))

	// // We may want to filter some headers, otherwise we could just use a shallow copy
	// // proxyReq.Header = req.Header
	// proxyReq.Header = make(http.Header)
	// for h, val := range req.Header {
	// 	proxyReq.Header[h] = val
	// }

	// resp, err := httpClient.Do(proxyReq)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadGateway)
	// 	return
	// }
	// defer resp.Body.Close()

	// // legacy code
}
