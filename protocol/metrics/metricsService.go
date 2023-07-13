package metrics

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/lavanet/lava/utils"
)

type AggregatedMetric struct {
	TotalLatency uint64
	RelaysCount  int64
	SuccessCount int64
}

type MetricService struct {
	AggregatedMetricMap *map[string]map[string]map[string]map[RelaySource]*AggregatedMetric
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
	metricChannelBufferSizeData := os.Getenv("METRICS_BUFFER_SIZE_NR")
	metricChannelBufferSize, _ := strconv.ParseInt(metricChannelBufferSizeData, 10, 64)
	mChannel := make(chan RelayMetrics, metricChannelBufferSize)
	result := &MetricService{
		MetricsChannel:      mChannel,
		ReportUrl:           reportMetricsUrl,
		AggregatedMetricMap: &map[string]map[string]map[string]map[RelaySource]*AggregatedMetric{},
	}

	// setup reader & sending of the results via http
	ticker := time.NewTicker(time.Duration(intervalForMetrics * time.Minute.Nanoseconds()))
	go func() {
		for {
			select {
			case <-ticker.C:
				{
					utils.LavaFormatInfo("metric triggered, sending accumulated data to server")
					result.SendEachProjectMetricData()
				}
			case metricData := <-mChannel:
				utils.LavaFormatInfo("reading from chanel data")
				result.storeAggregatedData(metricData)
			}
		}
	}()
	return result
}

func (m *MetricService) SendData(data RelayMetrics) {
	if m.MetricsChannel != nil {
		select {
		case m.MetricsChannel <- data:
		default:
			utils.LavaFormatInfo("channel is full, ignoring these data",
				utils.Attribute{Key: "projectHash", Value: data.ProjectHash},
				utils.Attribute{Key: "chainId", Value: data.ChainID},
				utils.Attribute{Key: "apiType", Value: data.APIType},
			)
		}
	}
}

func (m *MetricService) SendEachProjectMetricData() {
	if m.AggregatedMetricMap == nil {
		return
	}

	for projectKey, projectData := range *m.AggregatedMetricMap {
		toSendData := prepareArrayForProject(projectData, projectKey)
		go sendMetricsViaHttp(m.ReportUrl, toSendData)
	}
	// we reset to be ready for new metric data
	m.AggregatedMetricMap = &map[string]map[string]map[string]map[RelaySource]*AggregatedMetric{}
}

func prepareArrayForProject(projectData map[string]map[string]map[RelaySource]*AggregatedMetric, projectKey string) []RelayAnalyticsDTO {
	var toSendData []RelayAnalyticsDTO
	for chainKey, chainData := range projectData {
		for apiTypekey, apiTypeData := range chainData {
			for sourceKey, data := range apiTypeData {
				var averageLatency uint64
				if data.SuccessCount > 0 {
					averageLatency = data.TotalLatency / uint64(data.SuccessCount)
				}

				toSendData = append(toSendData, RelayAnalyticsDTO{
					ProjectHash:  projectKey,
					APIType:      apiTypekey,
					ChainID:      chainKey,
					Latency:      averageLatency,
					RelayCounts:  data.RelaysCount,
					SuccessCount: data.SuccessCount,
					Source:       sourceKey,
				})
			}
		}
	}
	return toSendData
}

func sendMetricsViaHttp(reportUrl string, data []RelayAnalyticsDTO) error {
	if len(data) == 0 {
		utils.LavaFormatDebug("no metrics found for this project.")
		return nil
	}
	jsonValue, err := json.Marshal(data)
	if err != nil {
		utils.LavaFormatError("error converting data to json", err)
		return err
	}
	resp, err := http.Post(reportUrl, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		utils.LavaFormatError("error posting data to report url.", err, utils.Attribute{Key: "url", Value: reportUrl})
		return err
	}
	if resp.StatusCode != http.StatusOK {
		utils.LavaFormatError("error status code returned from server.", nil, utils.Attribute{Key: "url", Value: reportUrl})
	}
	return nil
}

func (m *MetricService) storeAggregatedData(data RelayMetrics) error {
	utils.LavaFormatDebug("new data to store",
		utils.Attribute{Key: "projectHash", Value: data.ProjectHash},
		utils.Attribute{Key: "apiType", Value: data.APIType},
		utils.Attribute{Key: "chainId", Value: data.ChainID},
	)

	var successCount int64
	var successLatencyValue uint64
	if data.Success {
		successCount = 1
		successLatencyValue = uint64(data.Latency)
	}

	store := *m.AggregatedMetricMap // for simplicity during operations
	projectData, exists := store[data.ProjectHash]
	if exists {
		m.storeChainIdData(projectData, data, successCount, successLatencyValue)
	} else {
		// means we haven't stored any data yet for this project, so we build all the maps
		projectData = map[string]map[string]map[RelaySource]*AggregatedMetric{
			data.ChainID: {
				data.APIType: {
					data.Source: &AggregatedMetric{
						TotalLatency: successLatencyValue,
						RelaysCount:  1,
						SuccessCount: successCount,
					},
				},
			},
		}
		store[data.ProjectHash] = projectData
	}
	return nil
}

func (m *MetricService) storeChainIdData(projectData map[string]map[string]map[RelaySource]*AggregatedMetric, data RelayMetrics, successCount int64, successLatencyValue uint64) {
	chainIdData, exists := projectData[data.ChainID]
	if exists {
		m.storeApiTypeData(chainIdData, data, successCount, successLatencyValue)
	} else {
		chainIdData = map[string]map[RelaySource]*AggregatedMetric{
			data.APIType: {
				data.Source: &AggregatedMetric{
					TotalLatency: successLatencyValue,
					RelaysCount:  1,
					SuccessCount: successCount,
				},
			},
		}
		(*m.AggregatedMetricMap)[data.ProjectHash][data.ChainID] = chainIdData
	}
}

func (m *MetricService) storeApiTypeData(chainIdData map[string]map[RelaySource]*AggregatedMetric, data RelayMetrics, successCount int64, successLatencyValue uint64) {
	apiTypesData, exists := chainIdData[data.APIType]
	if exists {
		m.storeSourceData(apiTypesData, data, successCount, successLatencyValue)
	} else {
		(*m.AggregatedMetricMap)[data.ProjectHash][data.ChainID][data.APIType] = map[RelaySource]*AggregatedMetric{
			data.Source: {
				TotalLatency: successLatencyValue,
				RelaysCount:  1,
				SuccessCount: successCount,
			},
		}
	}
}

func (m *MetricService) storeSourceData(sourceData map[RelaySource]*AggregatedMetric, data RelayMetrics, successCount int64, successLatencyValue uint64) {
	existingData, exists := sourceData[data.Source]
	if exists {
		existingData.TotalLatency += successLatencyValue
		existingData.SuccessCount += successCount
		existingData.RelaysCount += 1
	} else {
		(*m.AggregatedMetricMap)[data.ProjectHash][data.ChainID][data.APIType][data.Source] = &AggregatedMetric{
			TotalLatency: successLatencyValue,
			RelaysCount:  1,
			SuccessCount: successCount,
		}
	}
}
