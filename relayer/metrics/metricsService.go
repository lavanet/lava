package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/lavanet/lava/utils"
	"net/http"
	"time"
)

type AggregatedMetric struct {
	TotalLatency int64
	RelaysCount  uint64
	SuccessCount int64
}

type MetricService struct {
	numberOfRelays      int64
	aggregatedMetricMap map[string]map[string]map[string]AggregatedMetric
	metricsChannel      chan RelayAnalytics
	sqsUrl              string
}

func NewMetricService(portalHttpSqsUrl string, intervalForSendingMetricsInM int) *MetricService {
	mChannel := make(chan RelayAnalytics)
	result := &MetricService{
		metricsChannel: mChannel,
		sqsUrl:         portalHttpSqsUrl,
	}
	//setup reader
	go result.ReadFromChannel()
	//setup sending of the results into the query
	//TODO update this with select
	for range time.Tick(time.Duration(intervalForSendingMetricsInM * 60 * 1000000000)) {
		go func() {
			data := result.ConvertDataToArray()
			err := result.SendTo(data)
			if err != nil {
				utils.LavaFormatError("error sending metrics data portal app.", err, nil)
			}
		}()
	}
	return result
}
func (m *MetricService) SendTo(data []RelayAnalyticsDTO) error {
	if len(data) > 0 {
		return nil
	}
	jsonValue, err := json.Marshal(data)
	if err != nil {
		return err
	}
	resp, err := http.Post(m.sqsUrl, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("couldn't send the data into the server")
	}
	return nil
}

func (m *MetricService) ConvertDataToArray() []RelayAnalyticsDTO {
	//project-chain-apiType
	toSendData := make([]RelayAnalyticsDTO, m.numberOfRelays)
	if m.numberOfRelays == 0 {
		return toSendData
	}
	for projectKey, projectData := range m.aggregatedMetricMap {
		for chainKey, chainData := range projectData {
			for apiTypekey, apiTypeData := range chainData {
				toSendData = append(toSendData, RelayAnalyticsDTO{
					ProjectHash:  projectKey,
					APIType:      apiTypekey,
					ChainID:      chainKey,
					Latency:      apiTypeData.TotalLatency,
					ComputeUnits: apiTypeData.RelaysCount,
					SuccessCount: apiTypeData.SuccessCount,
				})
			}
		}
	}
	m.aggregatedMetricMap = make(map[string]map[string]map[string]AggregatedMetric)
	m.numberOfRelays = 0
	return toSendData
}

func (m *MetricService) CloseChannel() {
	if m.metricsChannel != nil {
		m.CloseChannel()
	}
}

func (m *MetricService) SendDataToChannel(data RelayAnalytics) {
	if m.metricsChannel != nil {
		m.metricsChannel <- data
	}
}

func (m *MetricService) ReadFromChannel() {
	if m.metricsChannel != nil {
		for {
			data, newMessageInserted := <-m.metricsChannel
			if !newMessageInserted {
				//TODO check if this will work
				continue
			}
			toInsertData := AggregatedMetric{
				TotalLatency: data.Latency,
				RelaysCount:  data.ComputeUnits,
				SuccessCount: 0,
			}
			if data.Success {
				toInsertData.SuccessCount = 1
			}
			m.StoreAggregatedData(data.ProjectHash, data.ChainID, data.APIType, toInsertData)

		}
	}
}

func (m *MetricService) StoreAggregatedData(projectHash string, chainId string, apiType string, data AggregatedMetric) error {
	//TODO: refactor this for better readibility
	//project-chain-apiType
	projectData, exists := m.aggregatedMetricMap[projectHash]
	if !exists {
		projectData = map[string]map[string]AggregatedMetric{
			chainId: {
				apiType: data,
			},
		}
		m.numberOfRelays++
		m.aggregatedMetricMap[projectHash] = projectData
	} else {
		chainIdData, exists := projectData[chainId]
		if !exists {
			chainIdData = map[string]AggregatedMetric{
				chainId: data,
			}
			m.numberOfRelays++
			m.aggregatedMetricMap[projectHash][chainId] = chainIdData
		} else {
			apiTypesData, exists := chainIdData[apiType]
			if !exists {
				m.numberOfRelays++
				m.aggregatedMetricMap[projectHash][chainId][apiType] = data
			} else {
				apiTypesData.TotalLatency += data.TotalLatency
				apiTypesData.SuccessCount += data.SuccessCount
				apiTypesData.RelaysCount += data.RelaysCount
			}
		}
	}
	return nil
}
