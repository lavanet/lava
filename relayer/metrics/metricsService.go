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
	NumberOfRelays      int64
	AggregatedMetricMap *map[string]map[string]map[string]*AggregatedMetric
	MetricsChannel      chan RelayMetrics
	ReportUrl           string
}

func NewMetricService() *MetricService {
	reportMetricsUrl := os.Getenv("REPORT_METRICS_URL")
	intervalData := os.Getenv("METRICS_INTERVAL_FOR_SENDING_DATA_IN_M")
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
	// setup reader
	go result.readFromChannel()
	// setup sending of the results into the query
	ticker := time.NewTicker(time.Duration(intervalForMetrics * time.Minute.Nanoseconds()))
	go func() {
		for {
			select {
			case <-ticker.C:
				{
					data := result.ConvertDataToArray()
					err := result.SendTo(data)
					if err != nil {
						utils.LavaFormatError("error sending metrics data portal app.", err, nil)
					}
				}
			}
		}
	}()
	return result
}

func (m *MetricService) SendTo(data []RelayAnalyticsDTO) error {
	if data == nil || len(data) == 0 {
		return nil
	}
	jsonValue, err := json.Marshal(data)
	if err != nil {
		return err
	}
	resp, err := http.Post(m.ReportUrl, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("couldn't send the data into the server")
	}
	return nil
}

func (m *MetricService) ConvertDataToArray() []RelayAnalyticsDTO {
	if m.NumberOfRelays == 0 || m.AggregatedMetricMap == nil {
		return nil
	}
	toSendData := make([]RelayAnalyticsDTO, m.NumberOfRelays)
	counter := 0
	//project-chain-apiType
	for projectKey, projectData := range *m.AggregatedMetricMap {
		for chainKey, chainData := range projectData {
			for apiTypekey, apiTypeData := range chainData {
				toSendData[counter] = RelayAnalyticsDTO{
					ProjectHash:  projectKey,
					APIType:      apiTypekey,
					ChainID:      chainKey,
					Latency:      apiTypeData.TotalLatency / apiTypeData.RelaysCount,
					RelayCounts:  apiTypeData.RelaysCount,
					SuccessCount: apiTypeData.SuccessCount,
				}
				counter++
			}
		}
	}
	m.AggregatedMetricMap = &map[string]map[string]map[string]*AggregatedMetric{}
	m.NumberOfRelays = 0
	return toSendData
}

func (m *MetricService) SendData(data RelayMetrics) {
	if m.MetricsChannel != nil {
		m.MetricsChannel <- data
	}
}

func (m *MetricService) readFromChannel() {
	if m.MetricsChannel != nil {
		for {
			data, newMessageInserted := <-m.MetricsChannel
			if !newMessageInserted {
				continue
			}
			toInsertData := &AggregatedMetric{
				TotalLatency: data.Latency,
				RelaysCount:  1,
				SuccessCount: 0,
			}
			if data.Success {
				toInsertData.SuccessCount = 1
			}
			m.storeAggregatedData(data.ProjectHash, data.ChainID, data.APIType, toInsertData)

		}
	}
}

func (m *MetricService) storeAggregatedData(projectHash string, chainId string, apiType string, data *AggregatedMetric) error {
	//project-chain-apiType
	store := *m.AggregatedMetricMap
	projectData, exists := store[projectHash]
	if !exists {
		projectData = map[string]map[string]*AggregatedMetric{
			chainId: {
				apiType: data,
			},
		}
		m.NumberOfRelays = m.NumberOfRelays + 1
		store[projectHash] = projectData
	} else {
		chainIdData, exists := projectData[chainId]
		if !exists {
			chainIdData = map[string]*AggregatedMetric{
				chainId: data,
			}
			m.NumberOfRelays = m.NumberOfRelays + 1
			store[projectHash][chainId] = chainIdData
		} else {
			apiTypesData, exists := chainIdData[apiType]
			if !exists {
				m.NumberOfRelays = m.NumberOfRelays + 1
				store[projectHash][chainId][apiType] = data
			} else {
				apiTypesData.TotalLatency += data.TotalLatency
				apiTypesData.SuccessCount += data.SuccessCount
				apiTypesData.RelaysCount += data.RelaysCount
			}
		}
	}
	return nil
}
