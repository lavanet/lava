# Lava e2e Tests & Mock Proxy
### TL;DR
```
[+]$ export LAVA=$HOME/go/lava
[+]$ cd $LAVA && go test ./testutil/e2e/ -v      # Runs all Lava e2e Tests - Full Lava Node, Providers, Test_clients
[+]$ cd $LAVA && ./initFull.sh                   # Add specs, stakes, Runs all Proxies, Starts all Mock Providers (both eth&osmosis)
                                                 (Everything Excluding Lava & test_clients)
[+]$ cd $LAVA && ./lava.sh                       # Runs Everything, Lava, init, proxies, mock providers, test_clients
``` 


# Mock Proxy

### Start Proxies & Mock Providers 
```
# ETH
$ ./mock_proxy_eth.sh                   # will start proxies for eth
$ ./providers_eth_mock.sh               # will run eth providers directed towards the proxies

# OSMOSIS
$ ./mock_proxy_osmosis.sh               # will start proxies for osmosis
$ ./providers_osmosis_mock.sh           # will run osmosis providers directed towards the proxies
```

### Start & Use Generic Proxy Server
```
$ go run ./testutil/e2e/proxy/. [hostID] [port] [hostBaseURL] OPTIONAL -cache -mal

# for example
$ go run ./testutil/e2e/proxy/. random 1111 www.randomnumberapi.com              # will generate cache at random_1111.json
$ go run ./testutil/e2e/proxy/. random 1111 www.randomnumberapi.com -cache       # will load and reply from saved cache random_1111.json
$ go run ./testutil/e2e/proxy/. random 1111 www.randomnumberapi.com -cache -mal  # will load and reply from random_mal.json

# After starting the proxy server you can make any http request to it intead of the target host
# i.e
$ curl -v "0.0.0.0:1111/api/v1.0/random?min=100&max=1000&count=1"                # first time, this will return a random number
$ curl -v "0.0.0.0:1111/api/v1.0/random?min=100&max=1000&count=1"                # if -cache was set,
                                                                                     the second time, will return the SAME number
```

### Edit Cache & Mal Json
```
# Recorded Cache:
# json files are saved at
$LAVA/testutil/e2e/proxy/mockMaps/hostID_port.json

# change or add any key/value pair for request/response  -  they will be sent when -cache is set

# Mal (alternate) Cache:
# to make a mal (fake response) cache file, copy in the same directory the cache and change the _port.json to _mal.json - i.e:
$ cp ./testutil/e2e/proxy/mockMaps/random_1111.json ./testutil/e2e/proxy/mockMaps/random_mal.json 

# this file will be loaded instead of the original cache incase the -mal flag was set 
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