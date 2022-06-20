# Lava e2e Tests & Mock Proxy

<br>

### TL;DR
```
$ export LAVA=$HOME/go/lava
$ cd $LAVA && go test ./testutil/e2e/ -v      # Runs all Lava e2e Tests - Full Lava Node, Providers, Test_clients
$ cd $LAVA && ./initFull.sh                   # Add specs, stakes, Runs all Proxies, Starts all Mock Providers (both eth&osmosis)
                                                 (Everything Excluding Lava & test_clients)
$ cd $LAVA && ./lava.sh                       # Runs Everything, Lava, init, proxies, mock providers, test_clients
``` 


<br>

# Mock Proxy

### Start Proxies & Mock Providers 
```
# ETH
$ ./scripts/mock_proxy_eth.sh                   # will start proxies for eth
$ ./scripts/providers_eth_mock.sh               # will run eth providers directed towards the proxies

# OSMOSIS
$ ./scripts/mock_proxy_osmosis.sh               # will start proxies for osmosis
$ ./scripts/providers_osmosis_mock.sh           # will run osmosis providers directed towards the proxies
```

### Start & Use Generic Proxy Server
```
$ go run ./testutil/e2e/proxy/. [host] -p [port] OPTIONAL -cache -alt [JSONFILE] -id [hostID]

# for example
$ go run ./testutil/e2e/proxy/. randomnumberapi.com -p 1111 -id random                                  # will generate cache at ./mockMaps/random.json
$ go run ./testutil/e2e/proxy/. randomnumberapi.com -p 1111 -id random -cache                           # will load and reply from saved cache ./mockMaps/random.json
$ go run ./testutil/e2e/proxy/. randomnumberapi.com -p 1111 -cache -alt ./mockMaps/random_alt.json      # will load and reply from saved cache ./mockMaps/random_alt.json


# After starting the proxy server you can make any http request to it intead of the target host
# i.e
$ curl -v "0.0.0.0:1111/api/v1.0/random?min=100&max=1000&count=1"                # first time, this will return a random number
$ curl -v "0.0.0.0:1111/api/v1.0/random?min=100&max=1000&count=1"                # if -cache was set,
                                                                                     the second time, will return the SAME number

# Full Generic Mock Proxy Test
$ (sleep 1 && echo ' @@@ RANDOM NUMBER:' `curl "0.0.0.0:1111/api/v1.0/random?min=100&max=1000&count=1"` && \
     sleep 1 &&  echo ' @@@ RANDOM NUMBER:' `curl "0.0.0.0:1111/api/v1.0/random?min=100&max=1000&count=1"` && \
     echo '' && echo '@@@@@@@@@@@@@@@@@@@@@@@@' && \
     echo ' @@@ Test (-cache) Passed If The 2 Random Numbers WERE THE SAME' && echo "";) &  \
     go run ./testutil/e2e/proxy/. randomnumberapi.com -p 1111 -id random -cache 
```

### Edit Cache & Mal Json
```
# Recorded Cache:
# json files are saved at
$LAVA/testutil/e2e/proxy/mockMaps/hostID.json

# change or add any key/value pair for request/response  -  they will be sent when -cache is set

# Mal (alternate) Cache:
# to make a mal (fake response) cache file, copy in the same directory the cache and change the [hostID].json to [hostID]_alt.json - i.e:
$ cp ./testutil/e2e/proxy/mockMaps/random.json ./testutil/e2e/proxy/mockMaps/random_alt.json 

# this file will be loaded instead of the original cache incase the -alt flag was set 
```


<br>


# Lava e2e tests

### Run all Lava e2e Tests Locally
```
go test ./testutil/e2e/ -v -timeout 240s

# (you can also run the tests from vscode)
```

#### Logs will be saved to 
```
lava/testutil/e2e/logs/processID.log
```

#### Github Actions file for e2e Test
```
lava/.github/workflows/e2e.yml
```

#### Create a new test
```
- create a new file called some_test.go                      # must end with _test.go
    - make a function: func TestSomething(t *testing.T)      # must start with Test

# checkout simple_test.go or lava_fullflow_test.go
```


# TODO:
tldr
- initFull.sh + add portal

mock proxy
- TODO: add -cache -mal [id] arguments
- TODO: add -strict option ? return ONLY from cache
- TODO: combined cache + optional alternative file

<br><br><br>
## ~~~~~~~~~~~ maybe usefull ~~~~~~~~~~~~~~
### basic Proxy
```
func basicProxy(rw http.ResponseWriter, req *http.Request) {
	// All incoming requests will be sent to this host
	host := fmt.Sprintf("mainnet.infura.io")

	// Get request body
	rawBody := getDataFromIORead(&req.Body, true)
	println(" ::: INCOMING PROXY MSG :::", string(rawBody))

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
	// respBody, _ := ioutil.ReadAll(proxyRes.Body)
	respBody := getDataFromIORead(&proxyRes.Body, true)
	println(" ::: Real Response ::: ", string(respBody), "\n")

	// Change Response
	// respBody = []byte("{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xe000000000000000000\"}")
	// println(" ::: Fake Response ::: ", string(respBody))

	//Return Response
	returnResponse(rw, proxyRes.StatusCode, respBody)

}
```

### proxy docker experiment (Dockerfile & proxy.sh)
```
FROM nginx:1.15-alpine

COPY proxy.sh /usr/local/bin/

RUN apk add --update bash \
	&& rm -rf /var/cache/apk/* \
	&& chmod +x /usr/local/bin/proxy.sh

EXPOSE 80

CMD ["proxy.sh"]
```
```
#!/bin/bash
if [ -z "$REDIRECT_TYPE" ]; then
	REDIRECT_TYPE="permanent"
fi

if [ -z "$REDIRECT_TARGET" ]; then
	echo "Redirect target variable not set (REDIRECT_TARGET)"
	exit 1
else
	
	# Add trailing slash
	if [[ ${REDIRECT_TARGET:length-1:1} != "/" ]]; then
		REDIRECT_TARGET="$REDIRECT_TARGET/"
	fi
fi

# Default to 80
LISTEN="80"
# Listen to PORT variable given on Cloud Run Context
if [ ! -z "$PORT" ]; then
	LISTEN="$PORT"
fi


cat <<EOF > /etc/nginx/conf.d/default.conf

server {
	listen ${LISTEN};

	location "/" {                                        
		mirror "/mirror";                                                       
		mirror_request_body on;                                                 
		return 200;                                                             
	}                                                                           

	location = "/mirror" {                                                      
		internal;                                                               
		proxy_pass "https://mainnet.infura.io/v3/3755a1321ab24f938589412403c46455/";                   
		proxy_set_header Host "mainnet.infura.io";                            
		proxy_set_header X-Original-URI $request_uri;                           
		proxy_set_header X-SERVER-PORT $server_port;                            
		proxy_set_header X-SERVER-ADDR $server_addr;                            
		proxy_set_header X-REAL-IP $remote_addr;                                
	}
 
}
EOF

echo "Listening to $LISTEN, Redirecting HTTP requests to ${REDIRECT_TARGET}..."
exec nginx -g "daemon off;"

```

### Piping stdout/err into go
```
//simple_pipe.go
func mainA() {

	nBytes, nChunks := int64(0), int64(0)
	r := bufio.NewReader(os.Stdin)
	// buf := make([]byte, 0, 4*1024)
	num := 0
	for {
		line, err := r.ReadString('\n')
		fmt.Println(string("!!! nice !!! ") + string(line))
		num += 1
		if err != nil {
			fmt.Print(string("XXX error XXX ") + string(err.Error()))
		}

		if err == io.EOF {
			break
		}
	}

	fmt.Println("Bytes:", nBytes, "Chunks:", nChunks)
}
```