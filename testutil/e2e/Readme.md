# Lava e2e Tests & Mock Proxy
### TL;DR
```
$ export LAVA=$HOME/go/lava
$ cd $LAVA && go test ./testutil/e2e/ -v      # Runs all Lava e2e Tests - Full Lava Node, Providers, Test_clients
$ cd $LAVA && ./initFull.sh                   # Add specs, stakes, Runs all Proxies, Starts all Mock Providers (both eth&osmosis)
                                                 (Everything Excluding Lava & test_clients)
$ cd $LAVA && ./lava.sh                       # Runs Everything, Lava, init, proxies, mock providers, test_clients
``` 


# Mock Proxy

### Start Proxies & Mock Providers 
```
# ETH
$ ./.scripts/mock_proxy_eth.sh                   # will start proxies for eth
$ ./.scripts/providers_eth_mock.sh               # will run eth providers directed towards the proxies

# OSMOSIS
$ ./.scripts/mock_proxy_osmosis.sh               # will start proxies for osmosis
$ ./.scripts/providers_osmosis_mock.sh           # will run osmosis providers directed towards the proxies
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

# Full Generic Test
go run ./testutil/e2e/proxy/. randomnumberapi.com -p 1111 -cache -id random & sleep 1 &&  \
     curl -v "0.0.0.0:1111/api/v1.0/random?min=100&max=1000&count=1" && sleep 1 && \
     curl -v "0.0.0.0:1111/api/v1.0/random?min=100&max=1000&count=1"

(sleep 1 && echo ' @@@ RANDOM NUMBER:' `curl "0.0.0.0:1111/api/v1.0/random?min=100&max=1000&count=1"` && sleep 1 && \
     echo ' @@@ RANDOM NUMBER:' `curl "0.0.0.0:1111/api/v1.0/random?min=100&max=1000&count=1"` && \
     echo '' && echo '@@@@@@@@@@@@@@@@@@@@@@@@' && echo ' @@@ Test (-cache) Passed If The 2 Random Numbers WERE THE SAME' && echo "";) &  \
     go run ./testutil/e2e/proxy/. randomnumberapi.com -p 1111 -cache -id random 
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