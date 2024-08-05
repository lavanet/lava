package metrics

import (
	"bytes"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v2/utils"
)

type AggregatedMetric struct {
	TotalLatency uint64
	RelaysCount  int64
	SuccessCount int64
	TotalCu      uint64
	TimeStamp    time.Time
}

type MetricService struct {
	AggregatedMetricMap *map[string]map[string]map[string]map[RelaySource]map[string]*AggregatedMetric
	MetricsChannel      chan RelayMetrics
	ReportUrl           string
	ReportAuthorization string
}

func NewMetricService() *MetricService {
	reportMetricsUrl := os.Getenv("REPORT_METRICS_URL")
	reportMetricsAuthorization := os.Getenv("METRICS_AUTHORIZATION")
	intervalData := os.Getenv("METRICS_INTERVAL_FOR_SENDING_DATA_MIN")
	if reportMetricsUrl == "" || intervalData == "" {
		return nil
	}

	if reportMetricsAuthorization == "" {
		utils.LavaFormatInfo("Authorization is not set for metrics")
	}
	intervalForMetrics, _ := strconv.ParseInt(intervalData, 10, 32)
	metricChannelBufferSizeData := os.Getenv("METRICS_BUFFER_SIZE_NR")
	metricChannelBufferSize, _ := strconv.ParseInt(metricChannelBufferSizeData, 10, 64)
	mChannel := make(chan RelayMetrics, metricChannelBufferSize)
	result := &MetricService{
		MetricsChannel:      mChannel,
		ReportUrl:           reportMetricsUrl,
		ReportAuthorization: reportMetricsAuthorization,
		AggregatedMetricMap: &map[string]map[string]map[string]map[RelaySource]map[string]*AggregatedMetric{},
	}

	// setup reader & sending of the results via http
	ticker := time.NewTicker(time.Duration(intervalForMetrics * time.Minute.Nanoseconds()))
	go func() {
		for {
			select {
			case <-ticker.C:
				{
					utils.LavaFormatDebug("metric triggered, sending accumulated data to server")
					result.SendEachProjectMetricData()
				}
			case metricData := <-mChannel:
				utils.LavaFormatDebug("reading from chanel data")
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
			utils.LavaFormatDebug("channel is full, ignoring these data",
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
		go sendMetricsViaHttp(m.ReportUrl, m.ReportAuthorization, toSendData)
	}
	// we reset to be ready for new metric data
	m.AggregatedMetricMap = &map[string]map[string]map[string]map[RelaySource]map[string]*AggregatedMetric{}
}

func prepareArrayForProject(projectData map[string]map[string]map[RelaySource]map[string]*AggregatedMetric, projectKey string) []RelayAnalyticsDTO {
	var toSendData []RelayAnalyticsDTO
	for chainKey, chainData := range projectData {
		for apiTypekey, apiTypeData := range chainData {
			for sourceKey, sourceData := range apiTypeData {
				for origin, originData := range sourceData {
					var averageLatency uint64
					if originData.SuccessCount > 0 {
						averageLatency = originData.TotalLatency / uint64(originData.SuccessCount)
					}
					toSendData = append(toSendData, RelayAnalyticsDTO{
						ProjectHash:  projectKey,
						APIType:      apiTypekey,
						ChainID:      chainKey,
						Latency:      averageLatency,
						RelayCounts:  originData.RelaysCount,
						SuccessCount: originData.SuccessCount,
						Source:       sourceKey,
						TotalCu:      originData.TotalCu,
						Timestamp:    originData.TimeStamp.UTC().Format(time.RFC3339),
						Origin:       origin,
					})
				}
			}
		}
	}
	return toSendData
}

func sendMetricsViaHttp(reportUrl string, authorization string, data []RelayAnalyticsDTO) error {
	if len(data) == 0 {
		utils.LavaFormatDebug("no metrics found for this project.")
		return nil
	}
	jsonValue, err := json.Marshal(data)
	if err != nil {
		utils.LavaFormatError("error converting data to json", err)
		return err
	}
	req, err := http.NewRequest("POST", reportUrl, bytes.NewBuffer(jsonValue))
	if err != nil {
		utils.LavaFormatError("error creating request for metrics", err, utils.Attribute{Key: "url", Value: reportUrl})
		return err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
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
		projectData = map[string]map[string]map[RelaySource]map[string]*AggregatedMetric{
			data.ChainID: {
				data.APIType: {
					data.Source: {
						data.Origin: &AggregatedMetric{
							TotalLatency: successLatencyValue,
							RelaysCount:  1,
							SuccessCount: successCount,
							TimeStamp:    data.Timestamp,
							TotalCu:      data.ComputeUnits,
						},
					},
				},
			},
		}
		store[data.ProjectHash] = projectData
	}
	return nil
}

func (m *MetricService) storeChainIdData(projectData map[string]map[string]map[RelaySource]map[string]*AggregatedMetric, data RelayMetrics, successCount int64, successLatencyValue uint64) {
	chainIdData, exists := projectData[data.ChainID]
	if exists {
		m.storeApiTypeData(chainIdData, data, successCount, successLatencyValue)
	} else {
		chainIdData = map[string]map[RelaySource]map[string]*AggregatedMetric{
			data.APIType: {
				data.Source: {
					data.Origin: &AggregatedMetric{
						TotalLatency: successLatencyValue,
						RelaysCount:  1,
						SuccessCount: successCount,
						TimeStamp:    data.Timestamp,
						TotalCu:      data.ComputeUnits,
					},
				},
			},
		}
		(*m.AggregatedMetricMap)[data.ProjectHash][data.ChainID] = chainIdData
	}
}

func (m *MetricService) storeApiTypeData(chainIdData map[string]map[RelaySource]map[string]*AggregatedMetric, data RelayMetrics, successCount int64, successLatencyValue uint64) {
	apiTypesData, exists := chainIdData[data.APIType]
	if exists {
		m.storeSourceData(apiTypesData, data, successCount, successLatencyValue)
	} else {
		(*m.AggregatedMetricMap)[data.ProjectHash][data.ChainID][data.APIType] = map[RelaySource]map[string]*AggregatedMetric{
			data.Source: {
				data.Origin: {
					TotalLatency: successLatencyValue,
					RelaysCount:  1,
					SuccessCount: successCount,
					TimeStamp:    data.Timestamp,
					TotalCu:      data.ComputeUnits,
				},
			},
		}
	}
}

func (m *MetricService) storeSourceData(sourceData map[RelaySource]map[string]*AggregatedMetric, data RelayMetrics, successCount int64, successLatencyValue uint64) {
	existingData, exists := sourceData[data.Source]
	if exists {
		m.storeOriginData(existingData, data, successCount, successLatencyValue)
	} else {
		(*m.AggregatedMetricMap)[data.ProjectHash][data.ChainID][data.APIType][data.Source] = map[string]*AggregatedMetric{
			data.Origin: {
				TotalLatency: successLatencyValue,
				RelaysCount:  1,
				SuccessCount: successCount,
				TimeStamp:    data.Timestamp,
				TotalCu:      data.ComputeUnits,
			},
		}
	}
}

func (m *MetricService) storeOriginData(originData map[string]*AggregatedMetric, data RelayMetrics, successCount int64, successLatencyValue uint64) {
	existingData, exists := originData[data.Origin]
	if exists {
		existingData.TotalLatency += successLatencyValue
		existingData.SuccessCount += successCount
		existingData.TotalCu += data.ComputeUnits
		existingData.RelaysCount += 1
	} else {
		(*m.AggregatedMetricMap)[data.ProjectHash][data.ChainID][data.APIType][data.Source][data.Origin] = &AggregatedMetric{
			TotalLatency: successLatencyValue,
			RelaysCount:  1,
			SuccessCount: successCount,
			TimeStamp:    data.Timestamp,
			TotalCu:      data.ComputeUnits,
		}
	}
}
