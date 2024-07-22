package loadtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func startLoadTest(endpoint string, parallelClients int, callsPerClient int, method string, data string) {
	wg := sync.WaitGroup{}
	wg.Add(parallelClients)

	singleClient := func() {
		defer wg.Done()
		requestCreateErrors := 0
		failToSendErrors := 0
		non200StatusErrors := 0
		readBodyErrors := 0

		sendRequest := func() (*http.Response, error) {
			var body io.Reader
			if method == "POST" {
				body = bytes.NewBuffer([]byte(data))
			} else {
				body = nil
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
			if err != nil {
				utils.LavaFormatError("Failed to create request", err)
				requestCreateErrors++
				return nil, err
			}

			client := http.Client{
				Transport: &http.Transport{
					MaxIdleConnsPerHost: 10000,
				},
			}

			resp, err := client.Do(req)
			if err != nil {
				utils.LavaFormatDebug("Failed to send request", utils.LogAttr("error", err))
				failToSendErrors++
				return nil, err
			}

			return resp, nil
		}

		for j := 0; j < callsPerClient; j++ {
			resp, err := sendRequest()
			if err != nil {
				continue
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					utils.LavaFormatDebug("Failed to read response body", utils.LogAttr("error", err))
					readBodyErrors++
				}

				utils.LavaFormatDebug("Received non-200 status code",
					utils.LogAttr("statusCode", resp.StatusCode),
					utils.LogAttr("responseData", string(bodyBytes)),
				)

				non200StatusErrors++
				continue
			}
		}

		utils.LavaFormatInfo("Client finished",
			utils.LogAttr("failToSendErrorCount", failToSendErrors),
			utils.LogAttr("non200StatusErrorCount", non200StatusErrors),
			utils.LogAttr("readBodyErrorCount", readBodyErrors),
			utils.LogAttr("clientCreateErrorCount", requestCreateErrors),
		)
	}

	for i := 0; i < parallelClients; i++ {
		utils.LavaFormatDebug("Starting client", utils.LogAttr("client", i))
		go singleClient()
	}

	wg.Wait()
}

func CreateTestLoadCobraCommand() *cobra.Command {
	cmdTestLoad := &cobra.Command{
		Use:   `load <endpoint> <parallel clients> <calls per client> <method: GET/POST> [--data <data>]`,
		Short: `run a load test on the given endpoint, with the given number of parallel clients, and the given number of calls per client`,
		Example: `load "http://127.0.0.1:3361" 50 20 POST --data '{"jsonrpc":"2.0","method":"status","params":[],"id":"1"}'
load "http://127.0.0.1:3361/cosmos/base/tendermint/v1beta1/blocks/latest" 50 20 GET`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.RangeArgs(4, 5)(cmd, args); err != nil {
				return fmt.Errorf("invalid number of arguments, either its a single config file or repeated groups of 4 HOST:PORT chain-id api-interface [node_url,node_url_2], arg count: %d", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.LavaFormatInfo("Load Test started", utils.LogAttr("args", strings.Join(args, ";")))

			logLevel, err := cmd.Flags().GetString(flags.FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}

			utils.SetGlobalLoggingLevel(logLevel)

			endpoint := args[0]
			parallelClients := args[1]
			callsPerClient := args[2]
			method := args[3]
			data := ""
			if len(args) == 5 {
				data = args[4]
			} else if viper.IsSet("data") {
				data = viper.GetString("data")
				var dataJson map[string]interface{}
				err := json.Unmarshal([]byte(data), &dataJson)
				if err != nil {
					return fmt.Errorf("invalid data, must be a valid json object")
				}
			}

			if method != "GET" && method != "POST" {
				return fmt.Errorf("invalid method, only GET and POST are supported")
			}

			if method == "GET" && data != "" {
				return fmt.Errorf("GET method does not support data")
			}

			if method == "POST" && data == "" {
				return fmt.Errorf("POST method requires data")
			}

			parallelClientsInt, err := strconv.Atoi(parallelClients)
			if err != nil {
				return fmt.Errorf("parallel clients must be an integer")
			}

			if parallelClientsInt <= 0 {
				return fmt.Errorf("parallel clients must be greater than 0")
			}

			callsPerClientInt, err := strconv.Atoi(callsPerClient)
			if err != nil {
				return fmt.Errorf("calls per client must be an integer")
			}

			if callsPerClientInt <= 0 {
				return fmt.Errorf("calls per client must be greater than 0")
			}

			if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
				return fmt.Errorf("invalid endpoint, must start with http:// or https://")
			}

			startLoadTest(endpoint, parallelClientsInt, callsPerClientInt, method, data)
			return nil
		},
	}

	cmdTestLoad.Flags().String(flags.FlagLogLevel, "debug", "log level")
	cmdTestLoad.Flags().String("data", "", "The data to send with the request")
	return cmdTestLoad
}
