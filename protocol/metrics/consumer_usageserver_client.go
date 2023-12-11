package metrics

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/lavanet/lava/utils"
)

type ConsumerUsageserverClient struct {
	endPointAddress string
}

type UpdateMetricsRequest struct {
	RecordDate   string `json:"RecordDate"`
	Hash         string `json:"Hash"`
	Chain        string `json:"Chain"`
	ApiType      string `json:"ApiType"`
	RelaysInc    int    `json:"RelaysInc"`
	CuInc        int    `json:"CuInc"`
	LatencyToAdd int    `json:"LatencyToAdd"`
}

func NewConsumerUsageserverClient(endPointAddress string) *ConsumerUsageserverClient {
	if endPointAddress == DisabledFlagOption {
		utils.LavaFormatDebug("Consumer Usageserver disabled")
		return nil
	}
	utils.LavaFormatDebug("Consumer Usageserver enabled", utils.Attribute{Key: "endPointAddress", Value: endPointAddress})
	return &ConsumerUsageserverClient{
		endPointAddress: endPointAddress,
	}
}

func (cuc *ConsumerUsageserverClient) SetRelayMetrics(relayMetric *RelayMetrics, err error) error {
	// TODO: run in go in the background , and agregate

	if cuc == nil {
		return nil
	}
	updateMetricsRequest := UpdateMetricsRequest{
		RecordDate:   relayMetric.Timestamp.Format("20060102"),
		Hash:         relayMetric.ProjectHash,
		Chain:        relayMetric.ChainID,
		ApiType:      relayMetric.APIType,
		CuInc:        int(relayMetric.ComputeUnits),
		LatencyToAdd: int(relayMetric.Latency),
		RelaysInc:    1,
	}

	jsonData, err := json.Marshal(updateMetricsRequest)
	if err != nil {
		utils.LavaFormatError("Failed to marshal UpdateMetricsRequest", err)
		return err
	}

	resp, err := http.Post(cuc.endPointAddress+"/updateMetrics", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		utils.LavaFormatError("Failed to post UpdateMetricsRequest", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.LavaFormatError("Failed to read response body", err)
		return err
	}

	utils.LavaFormatDebug("Consumer Usageserver relay submited", utils.Attribute{Key: "JsonResponse", Value: body})

	return nil
}
