package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
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
	sendCnt            int
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
		utils.LavaFormatInfo("Running with Consumer Relay Server Disabled")
		return nil
	}

	utils.LavaFormatInfo("[CUC] CUC is enabled", utils.LogAttr("endPointAddress", endPointAddress))

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
		return
	}
	utils.LavaFormatDebug("[CUC] Starting relayDataSendQueueStart loop")
	for range cuc.ticker.C {
		cuc.relayDataSendQueueTick()
	}
}

func (cuc *ConsumerRelayServerClient) relayDataSendQueueTick() {
	if cuc == nil {
		return
	}

	cuc.lock.Lock()
	defer cuc.lock.Unlock()

	if !cuc.isSendQueueRunning && len(cuc.addQueue) > 0 {
		sendQueue := cuc.addQueue
		cuc.addQueue = make([]UpdateMetricsRequest, 0)
		cuc.isSendQueueRunning = true
		cuc.sendCnt++
		utils.LavaFormatDebug("[CUC] Swapped queues", utils.LogAttr("sendQueue_length", len((sendQueue))), utils.LogAttr("sendcnt", cuc.sendCnt))

		sendCnt := cuc.sendCnt
		cucEndpointAddress := cuc.endPointAddress

		go func() {
			cuc.sendRelayData(sendQueue, sendCnt, cucEndpointAddress)

			cuc.lock.Lock()
			cuc.isSendQueueRunning = false
			cuc.lock.Unlock()
		}()
	} else {
		utils.LavaFormatDebug("[CUC] Sending in progress, skipping iteration will send on next", utils.LogAttr("sendcnt", cuc.sendCnt))
	}
}

func (cuc *ConsumerRelayServerClient) appendQueue(request UpdateMetricsRequest) {
	cuc.lock.Lock()
	defer cuc.lock.Unlock()
	cuc.addQueue = append(cuc.addQueue, request)
}

func (cuc *ConsumerRelayServerClient) SetRelayMetrics(relayMetric *RelayMetrics) {
	if cuc == nil {
		return
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
	cuc.appendQueue(request)
}

func (cuc *ConsumerRelayServerClient) aggregateAndSendRelayData(sendQueue []UpdateMetricsRequest, sendCnt int, cucEndpointAddress string) (*http.Response, error) {
	utils.LavaFormatDebug("[CUC] Sending data to server - start ", utils.LogAttr("Size of metrics to send", len(sendQueue)))

	if cuc == nil {
		return nil, utils.LavaFormatError("sending data to server - CUC is nil. misuse detected", nil)
	}

	if len(sendQueue) == 0 {
		return nil, errors.New("sending data to server - SendQueue is empty")
	}

	aggregatedRequests := cuc.aggregateRelayData(sendQueue)

	if len(aggregatedRequests) == 0 {
		return nil, errors.New("sending data to server - No requests after aggregate")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	jsonData, err := json.Marshal(aggregatedRequests)
	if err != nil {
		return nil, utils.LavaFormatError("sending data to server - Failed marshaling aggregated requests", err)
	}

	var resp *http.Response
	for i := 0; i < 3; i++ {
		resp, err = client.Post(cucEndpointAddress+"/updateMetrics", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			utils.LavaFormatDebug("[CUC] Sending data to server - Failed to post request", utils.LogAttr("Attempt", i+1), utils.LogAttr("err", err))
			time.Sleep(2 * time.Second)
		} else {
			utils.LavaFormatDebug("[CUC] Sending data to server - Successfully sent request", utils.LogAttr("Attempt", i+1), utils.LogAttr("Number of aggregated requests", len(aggregatedRequests)))
			return resp, nil
		}
	}

	return nil, utils.LavaFormatWarning("[CUC] Sending data to server - Failed to send requests after 3 attempts", err)
}

func (cuc *ConsumerRelayServerClient) handleSendRelayResponse(resp *http.Response, sendCnt int) {
	if cuc == nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			utils.LavaFormatWarning("[CUC] Sending data to server - failed reading response body", err)
		} else {
			utils.LavaFormatWarning("[CUC] Sending data to server - Received non-200 status code", nil, utils.LogAttr("status_code", resp.StatusCode), utils.LogAttr("body", string(bodyBytes)))
		}
	}
}

func (cuc *ConsumerRelayServerClient) sendRelayData(sendQueue []UpdateMetricsRequest, sendCnt int, cucEndpointAddress string) {
	if cuc == nil {
		return
	}
	resp, err := cuc.aggregateAndSendRelayData(sendQueue, sendCnt, cucEndpointAddress)
	if err != nil {
		utils.LavaFormatWarning("[CUC] Sending data to server - failed sendRelay data", err)
		return
	}
	cuc.handleSendRelayResponse(resp, sendCnt)
}

func generateRequestArregatedCacheKey(req UpdateMetricsRequest) string {
	return req.RecordDate + req.Hash + req.Chain + req.ApiType
}

func (cuc *ConsumerRelayServerClient) aggregateRelayData(reqs []UpdateMetricsRequest) []UpdateMetricsRequest {
	if cuc == nil {
		return nil
	}

	// Create a map to hold the aggregated data
	aggregated := make(map[string]*UpdateMetricsRequest)

	for _, req := range reqs {
		// Use the combination of RecordDate, Hash, Chain, and ApiType as the key
		key := generateRequestArregatedCacheKey(req)

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
