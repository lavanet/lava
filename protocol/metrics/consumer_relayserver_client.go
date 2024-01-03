package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lavanet/lava/utils"
)

type ConsumerRelayServerClient struct {
	endPointAddress    string
	addQueue           []UpdateMetricsRequest
	sendQueue          []UpdateMetricsRequest
	ticker             *time.Ticker
	lock               sync.RWMutex
	sendID             int
	isSendQueueRunning bool
}
type UpdateMetricsRequest struct {
	RecordDate      string `json:"RecordDate"`
	Hash            string `json:"Hash"`
	Chain           string `json:"Chain"`
	ApiType         string `json:"ApiType"`
	RelaysInc       uint64 `json:"RelaysInc"`
	CuInc           int    `json:"CuInc"`
	LatencyToAdd    uint64 `json:"LatencyToAdd"`
	LatencyAvgCount int    `json:"LatencyAvgCount"`
}

func NewConsumerRelayServerClient(endPointAddress string) *ConsumerRelayServerClient {
	if endPointAddress == DisabledFlagOption {
		utils.LavaFormatDebug("CUC: Consumer Usageserver disabled")
		return nil
	}

	cuc := &ConsumerRelayServerClient{
		endPointAddress: endPointAddress,
		ticker:          time.NewTicker(30 * time.Second),
		sendQueue:       make([]UpdateMetricsRequest, 0),
		addQueue:        make([]UpdateMetricsRequest, 0),
		lock:            sync.RWMutex{},
	}

	utils.LavaFormatDebug("CUC: Starting processQueue goroutine")
	go cuc.processQueue()

	return cuc
}

func (cuc *ConsumerRelayServerClient) processQueue() {
	if cuc == nil {
		return
	}

	for range cuc.ticker.C {
		cuc.lock.Lock()
		if !cuc.isSendQueueRunning && len(cuc.addQueue) > 0 {
			cuc.sendQueue, cuc.addQueue = cuc.addQueue, make([]UpdateMetricsRequest, 0)
			cuc.isSendQueueRunning = true
			cuc.sendID++
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Swapped queues (sendQueue length: %d) and set isSendQueueRunning to true - CCC iter:%d.", len(cuc.sendQueue), cuc.sendID))
			cuc.lock.Unlock()

			cuc.lock.RLock()
			sendQueueLength := len(cuc.sendQueue)
			sendID := cuc.sendID
			sendQueue := cuc.sendQueue
			cuc.lock.RUnlock()

			utils.LavaFormatDebug(fmt.Sprintf("CUC: Sending requests (sendQueue length: %d) - async start - CCC iter:%d.", sendQueueLength, sendID))
			cuc.sendRequests(sendQueue)
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Sending requests - async end - CCC iter:%d.", sendID))

			cuc.lock.Lock()
			cuc.isSendQueueRunning = false
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Finished sending requests and set isSendQueueRunning to false - CCC iter:%d.", cuc.sendID))
			cuc.lock.Unlock()
		} else {
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Skipped this iteration - CCC iter:%d.", cuc.sendID))
			cuc.lock.Unlock()
		}
	}
}

func (cuc *ConsumerRelayServerClient) SetRelayMetrics(relayMetric *RelayMetrics) error {
	if cuc == nil {
		return nil
	}

	request := UpdateMetricsRequest{
		RecordDate:      relayMetric.Timestamp.Format("20060102"),
		Hash:            relayMetric.ProjectHash,
		Chain:           relayMetric.ChainID,
		ApiType:         relayMetric.APIType,
		RelaysInc:       1,
		CuInc:           int(relayMetric.ComputeUnits),
		LatencyToAdd:    uint64(relayMetric.Latency),
		LatencyAvgCount: 1,
	}

	cuc.lock.Lock()
	cuc.addQueue = append(cuc.addQueue, request)
	cuc.lock.Unlock()

	return nil
}

func (cuc *ConsumerRelayServerClient) sendRequests(sendQueue []UpdateMetricsRequest) {
	if cuc == nil {
		return
	}

	utils.LavaFormatDebug("CUC: Starting sendRequests")

	if len(sendQueue) == 0 {
		utils.LavaFormatDebug("CUC: sendQueue is nil or empty")
		utils.LavaFormatDebug("CUC: sendQueue", utils.Attribute{Key: "Value", Value: sendQueue}, utils.Attribute{Key: "Length", Value: len(sendQueue)})
		return
	}

	aggregatedRequests := cuc.aggregateRequests(sendQueue)

	if len(aggregatedRequests) == 0 {
		utils.LavaFormatDebug("CUC: No requests after aggregate")
		utils.LavaFormatDebug("CUC: sendQueue", utils.Attribute{Key: "Value", Value: sendQueue}, utils.Attribute{Key: "Length", Value: len(sendQueue)})
		utils.LavaFormatDebug("CUC: aggregatedRequests", utils.Attribute{Key: "Value", Value: aggregatedRequests}, utils.Attribute{Key: "Length", Value: len(aggregatedRequests)})
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	jsonData, err := json.Marshal(aggregatedRequests)
	if err != nil {
		utils.LavaFormatDebug("CUC: Error marshalling sendQueue to JSON")
		return
	}

	cuc.lock.RLock()
	var cucEndpointAddress string = cuc.endPointAddress
	cuc.lock.RUnlock()

	var resp *http.Response
	for i := 0; i < 3; i++ {
		utils.LavaFormatDebug(fmt.Sprintf("CUC: Attempting to post request - Attempt: %d", i+1))
		resp, err = client.Post(cucEndpointAddress+"/updateMetrics", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Failed to post request - Attempt: %d, Error: %s", i+1, err.Error()))
			time.Sleep(2 * time.Second)
		} else if resp.StatusCode != http.StatusOK {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				utils.LavaFormatDebug(fmt.Sprintf("CUC: Error reading response body: %s", err.Error()))
			} else {
				bodyString := string(bodyBytes)
				utils.LavaFormatDebug(fmt.Sprintf("CUC: Received non-200 status code - Attempt: %d, Status code: %d, Response: %s", i+1, resp.StatusCode, bodyString))
			}
		} else {
			break
		}
	}

	if err != nil {
		utils.LavaFormatDebug("CUC: Failed to send requests after 3 attempts")
		return
	}

	defer resp.Body.Close()

	cuc.lock.RLock()
	utils.LavaFormatDebug(fmt.Sprintf("CUC: Successfully sent requests - CCC iter:%d.", cuc.sendID))
	cuc.lock.RUnlock()

}

func (cuc *ConsumerRelayServerClient) aggregateRequests(reqs []UpdateMetricsRequest) []UpdateMetricsRequest {
	if cuc == nil {
		return nil
	}

	// Create a map to hold the aggregated data
	aggregated := make(map[string]*UpdateMetricsRequest)

	for _, req := range reqs {
		// Use the combination of RecordDate, Hash, Chain, and ApiType as the key
		key := req.RecordDate + req.Hash + req.Chain + req.ApiType

		// If the key doesn't exist in the map, add it
		if _, exists := aggregated[key]; !exists {
			aggregated[key] = &UpdateMetricsRequest{
				RecordDate: req.RecordDate,
				Hash:       req.Hash,
				Chain:      req.Chain,
				ApiType:    req.ApiType,
			}
		}

		// Aggregate the data
		aggregated[key].RelaysInc += req.RelaysInc
		aggregated[key].CuInc += req.CuInc
		aggregated[key].LatencyToAdd += req.LatencyToAdd
		aggregated[key].LatencyAvgCount += req.LatencyAvgCount
	}

	// Convert the map to a slice
	var aggregatedSlice []UpdateMetricsRequest
	for _, req := range aggregated {
		aggregatedSlice = append(aggregatedSlice, *req)
	}

	return aggregatedSlice
}
