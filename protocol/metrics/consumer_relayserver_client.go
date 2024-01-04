package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
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
		addQueue:        make([]UpdateMetricsRequest, 0),
	}

	go cuc.relayDataSendQueueStart()

	return cuc
}

func (cuc *ConsumerRelayServerClient) relayDataSendQueueStart() {
	if cuc == nil {
		utils.LavaFormatWarning("CUC: Warning - CUC is nil. called from relayDataSendQueueStart", nil)
		return
	}

	utils.LavaFormatDebug("CUC: Starting relayDataSendQueueStart loop")

	for range cuc.ticker.C {
		cuc.relayDataSendQueueTick()
	}
}

func (cuc *ConsumerRelayServerClient) relayDataSendQueueTick() {
	if cuc == nil {
		utils.LavaFormatWarning("CUC: Warning - CUC is nil. called from relayDataSendQueueTick", nil)
		return
	}

	cuc.lock.Lock()
	defer cuc.lock.Unlock()

	if !cuc.isSendQueueRunning && len(cuc.addQueue) > 0 {
		sendQueue := cuc.addQueue
		cuc.addQueue = make([]UpdateMetricsRequest, 0)
		cuc.isSendQueueRunning = true
		cuc.sendID++
		utils.LavaFormatDebug(fmt.Sprintf("CUC: Swapped queues (sendQueue length: %d) and set isSendQueueRunning to true - CCC iter:%d.", len(sendQueue), cuc.sendID))

		sendID := cuc.sendID
		cucEndpointAddress := cuc.endPointAddress

		go func() {
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Sending requests (sendQueue length: %d) - async start - CCC iter:%d.", len(sendQueue), sendID))
			cuc.sendRelayData(sendQueue, sendID, cucEndpointAddress)
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Sending requests - async end - CCC iter:%d.", sendID))

			cuc.lock.Lock()
			cuc.isSendQueueRunning = false
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Finished sending requests and set isSendQueueRunning to false - CCC iter:%d.", cuc.sendID))
			cuc.lock.Unlock()
		}()
	} else {
		utils.LavaFormatDebug(fmt.Sprintf("CUC: Skipped this iteration - CCC iter:%d.", cuc.sendID))
	}
}

func (cuc *ConsumerRelayServerClient) SetRelayMetrics(relayMetric *RelayMetrics) error {
	if cuc == nil {
		utils.LavaFormatWarning("CUC: Warning - CUC is nil. called from SetRelayMetrics", nil)
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

func (cuc *ConsumerRelayServerClient) agregateAndSendRelayData(sendQueue []UpdateMetricsRequest, sendID int, cucEndpointAddress string) (*http.Response, error) {
	if cuc == nil {
		utils.LavaFormatWarning("CUC: Warning - CUC is nil. called from agregateAndSendRelayData", nil)
		return nil, errors.New("CUC is nil")
	}

	if len(sendQueue) == 0 {
		utils.LavaFormatDebug("CUC: sendQueue is nil or empty")
		return nil, errors.New("sendQueue is nil or empty")
	}

	aggregatedRequests := cuc.aggregateRelayData(sendQueue)

	if len(aggregatedRequests) == 0 {
		utils.LavaFormatDebug("CUC: No requests after aggregate")
		return nil, errors.New("No requests after aggregate")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	jsonData, err := json.Marshal(aggregatedRequests)
	if err != nil {
		utils.LavaFormatDebug("CUC: Error marshalling sendQueue to JSON")
		return nil, err
	}

	var resp *http.Response
	for i := 0; i < 3; i++ {
		utils.LavaFormatDebug(fmt.Sprintf("CUC: Attempting to post request - Attempt: %d", i+1))
		resp, err = client.Post(cucEndpointAddress+"/updateMetrics", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Failed to post request - Attempt: %d, Error: %s", i+1, err.Error()))
			time.Sleep(2 * time.Second)
		} else {
			return resp, nil
		}
	}

	return nil, errors.New("CUC: Failed to send requests after 3 attempts")
}

func (cuc *ConsumerRelayServerClient) handleSendRelayResponse(resp *http.Response, sendID int) {
	if cuc == nil {
		utils.LavaFormatWarning("CUC: Warning - CUC is nil. called from handleSendRelayResponse", nil)
		return
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Error reading response body: %s", err.Error()))
		} else {
			bodyString := string(bodyBytes)
			utils.LavaFormatDebug(fmt.Sprintf("CUC: Received non-200 status code - Status code: %d, Response: %s", resp.StatusCode, bodyString))
		}
	}

	defer resp.Body.Close()

	utils.LavaFormatDebug(fmt.Sprintf("CUC: Sending requests - async end - CCC iter:%d.", sendID))
}

func (cuc *ConsumerRelayServerClient) sendRelayData(sendQueue []UpdateMetricsRequest, sendID int, cucEndpointAddress string) {
	if cuc == nil {
		utils.LavaFormatWarning("CUC: Warning - CUC is nil. called from sendRelayData", nil)
		return
	}

	resp, err := cuc.agregateAndSendRelayData(sendQueue, sendID, cucEndpointAddress)
	if err != nil {
		utils.LavaFormatDebug(fmt.Sprintf("CUC: %s", err.Error()))
		return
	}

	cuc.handleSendRelayResponse(resp, sendID)
}

func (cuc *ConsumerRelayServerClient) aggregateRelayData(reqs []UpdateMetricsRequest) []UpdateMetricsRequest {
	if cuc == nil {
		utils.LavaFormatWarning("CUC: Warning - CUC is nil. called from aggregateRelayData", nil)
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
