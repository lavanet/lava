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
	RelaysCount  uint64
	SuccessCount int64
}

type MetricService struct {
	numberOfRelays      int64
	aggregatedMetricMap *map[string]map[string]map[string]*AggregatedMetric
	metricsChannel      chan RelayAnalytics
	sqsUrl              string
}

func NewMetricService() *MetricService {
	metricPortalSqsUrl := os.Getenv("METRICS_PORTAL_SQS_URL")
	intervalData := os.Getenv("METRICS_INTERVAL_FOR_SENDING_DATA_INM")
	if metricPortalSqsUrl == "" || intervalData == "" {
		return nil
	}
	intervalForMetrics, _ := strconv.ParseInt(intervalData, 10, 32)
	utils.LavaFormatDebug("env variables:", &map[string]string{
		"url":      metricPortalSqsUrl,
		"interval": intervalData})
	mChannel := make(chan RelayAnalytics)
	result := &MetricService{
		metricsChannel:      mChannel,
		sqsUrl:              metricPortalSqsUrl,
		aggregatedMetricMap: &map[string]map[string]map[string]*AggregatedMetric{},
	}
	//setup reader
	go result.ReadFromChannel()
	//setup sending of the results into the query
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
	if m.numberOfRelays == 0 || m.aggregatedMetricMap == nil {
		return nil
	}
	toSendData := make([]RelayAnalyticsDTO, m.numberOfRelays)
	counter := 0
	//project-chain-apiType
	for projectKey, projectData := range *m.aggregatedMetricMap {
		for chainKey, chainData := range projectData {
			for apiTypekey, apiTypeData := range chainData {
				toSendData[counter] = RelayAnalyticsDTO{
					ProjectHash:  projectKey,
					APIType:      apiTypekey,
					ChainID:      chainKey,
					Latency:      apiTypeData.TotalLatency,
					ComputeUnits: apiTypeData.RelaysCount,
					SuccessCount: apiTypeData.SuccessCount,
				}
				counter++
			}
		}
	}
	m.aggregatedMetricMap = &map[string]map[string]map[string]*AggregatedMetric{}
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
				continue
			}
			toInsertData := &AggregatedMetric{
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

func (m *MetricService) StoreAggregatedData(projectHash string, chainId string, apiType string, data *AggregatedMetric) error {
	//project-chain-apiType
	store := *m.aggregatedMetricMap
	projectData, exists := store[projectHash]
	if !exists {
		projectData = map[string]map[string]*AggregatedMetric{
			chainId: {
				apiType: data,
			},
		}
		m.numberOfRelays = m.numberOfRelays + 1
		store[projectHash] = projectData
	} else {
		chainIdData, exists := projectData[chainId]
		if !exists {
			chainIdData = map[string]*AggregatedMetric{
				chainId: data,
			}
			m.numberOfRelays = m.numberOfRelays + 1
			store[projectHash][chainId] = chainIdData
		} else {
			apiTypesData, exists := chainIdData[apiType]
			if !exists {
				m.numberOfRelays = m.numberOfRelays + 1
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
