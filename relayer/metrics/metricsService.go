package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/lavanet/lava/utils"
	"net/http"
	"os"
	"strconv"
	"time"
)

type AggregatedMetric struct {
	TotalLatency int64
	RelaysCount  int64
	SuccessCount int64
}

type MetricService struct {
	AggregatedMetricMap *map[string]map[string]map[string]*AggregatedMetric
	MetricsChannel      chan RelayMetrics
	ReportUrl           string
}

func NewMetricService() *MetricService {
	reportMetricsUrl := os.Getenv("REPORT_METRICS_URL")
	intervalData := os.Getenv("METRICS_INTERVAL_FOR_SENDING_DATA_MIN")
	if reportMetricsUrl == "" || intervalData == "" {
		return nil
	}
	intervalForMetrics, _ := strconv.ParseInt(intervalData, 10, 32)
	mChannel := make(chan RelayMetrics)
	result := &MetricService{
		MetricsChannel:      mChannel,
		ReportUrl:           reportMetricsUrl,
		AggregatedMetricMap: &map[string]map[string]map[string]*AggregatedMetric{},
	}

	// setup reader & sending of the results via http
	ticker := time.NewTicker(time.Duration(intervalForMetrics * time.Minute.Nanoseconds()))
	go func() {
		for {
			select {
			case <-ticker.C:
				{
					utils.LavaFormatInfo("metric triggered, sending accumulated data to server", nil)
					result.SendEachProjectMetricData()
				}
			}
			select {
			case metricData := <-mChannel:
				result.storeAggregatedData(metricData)
			}
		}
	}()
	return result
}

func (m *MetricService) SendData(data RelayMetrics) {
	if m.MetricsChannel != nil {
		m.MetricsChannel <- data
	}
}

func (m *MetricService) SendEachProjectMetricData() {
	if m.AggregatedMetricMap == nil {
		return
	}

	for projectKey, projectData := range *m.AggregatedMetricMap {
		toSendData := prepareArrayForProject(projectData, projectKey)
		err := sendMetricsViaHttp(m.ReportUrl, toSendData)
		if err != nil {
			utils.LavaFormatError("error sending project metrics data", err, &map[string]string{
				"projectHash": projectKey,
			})
		}
	}
	// we reset to be ready for new metric data
	m.AggregatedMetricMap = &map[string]map[string]map[string]*AggregatedMetric{}
	return
}

func prepareArrayForProject(projectData map[string]map[string]*AggregatedMetric, projectKey string) []RelayAnalyticsDTO {
	var toSendData []RelayAnalyticsDTO
	for chainKey, chainData := range projectData {
		for apiTypekey, apiTypeData := range chainData {
			toSendData = append(toSendData, RelayAnalyticsDTO{
				ProjectHash:  projectKey,
				APIType:      apiTypekey,
				ChainID:      chainKey,
				Latency:      apiTypeData.TotalLatency / apiTypeData.RelaysCount, // we loose the precise during this, and this would never be 0 if we have any record on this project
				RelayCounts:  apiTypeData.RelaysCount,
				SuccessCount: apiTypeData.SuccessCount,
			})
		}
	}
	return toSendData
}

func sendMetricsViaHttp(reportUrl string, data []RelayAnalyticsDTO) error {
	if data == nil || len(data) == 0 {
		utils.LavaFormatDebug("no metrics found for this project.", nil)
		return nil
	}
	jsonValue, err := json.Marshal(data)
	if err != nil {
		return err
	}
	resp, err := http.Post(reportUrl, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("couldn't send the data into the server")
	}
	return nil
}

func (m *MetricService) storeAggregatedData(data RelayMetrics) error {
	utils.LavaFormatDebug("new data to store", &map[string]string{
		"projectHash": data.ProjectHash,
		"apiType":     data.APIType,
		"chainId":     data.ChainID,
	})

	var successCount int64
	if data.Success {
		successCount = 1
	}

	store := *m.AggregatedMetricMap // for simplicity during operations
	projectData, exists := store[data.ProjectHash]
	if !exists {
		// means we haven't stored any data yet for this project, so we build all the maps
		projectData = map[string]map[string]*AggregatedMetric{
			data.ChainID: {
				data.APIType: &AggregatedMetric{
					TotalLatency: data.Latency,
					RelaysCount:  1,
					SuccessCount: successCount,
				},
			},
		}
		store[data.ProjectHash] = projectData
	} else {
		m.storeChainIdData(projectData, data, successCount)
	}
	return nil
}

func (m *MetricService) storeChainIdData(projectData map[string]map[string]*AggregatedMetric, data RelayMetrics, successCount int64) {
	chainIdData, exists := projectData[data.ChainID]
	if !exists {
		chainIdData = map[string]*AggregatedMetric{
			data.ChainID: &AggregatedMetric{
				TotalLatency: data.Latency,
				RelaysCount:  1,
				SuccessCount: successCount,
			},
		}
		(*m.AggregatedMetricMap)[data.ProjectHash][data.ChainID] = chainIdData
	} else {
		m.storeApiTypeData(chainIdData, data, successCount)
	}
}

func (m *MetricService) storeApiTypeData(chainIdData map[string]*AggregatedMetric, data RelayMetrics, successCount int64) {
	apiTypesData, exists := chainIdData[data.APIType]
	if !exists {
		(*m.AggregatedMetricMap)[data.ProjectHash][data.ChainID][data.APIType] = &AggregatedMetric{
			TotalLatency: data.Latency,
			RelaysCount:  1,
			SuccessCount: successCount,
		}
	} else {
		apiTypesData.TotalLatency += data.Latency
		apiTypesData.SuccessCount += successCount
		apiTypesData.RelaysCount += 1
	}
}
