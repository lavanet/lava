package metrics

import (
	"fmt"
	"testing"
)

func Test_StoreAggregatedData_OnMetricService(t *testing.T) {
	// setup
	metricService := MetricService{
		AggregatedMetricMap: &map[string]map[string]map[string]map[RelaySource]map[string]*AggregatedMetric{},
	}
	metricData := RelayMetrics{
		ProjectHash: "1",
		ChainID:     "testChain",
		APIType:     "testApiType",
		Success:     true,
		Latency:     50,
		Source:      GatewaySource,
		Origin:      "origin",
	}
	expectedMetricData := RelayAnalyticsDTO{
		ProjectHash:  "1",
		ChainID:      "testChain",
		APIType:      "testApiType",
		SuccessCount: 1,
		Latency:      50,
		RelayCounts:  1,
		Source:       GatewaySource,
		Origin:       "origin",
	}
	t.Run("SuccessRelay_EmptyMap", func(t *testing.T) {
		// arrange
		metricService.storeAggregatedData(metricData)

		// assertion
		err := checkThatMetricDtoInAggregatedMetricMap(*metricService.AggregatedMetricMap, expectedMetricData)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("SuccessRelay_NonEmptyMap", func(t *testing.T) {
		expectedMetricData = RelayAnalyticsDTO{
			ProjectHash:  "1",
			ChainID:      "testChain",
			APIType:      "testApiType",
			SuccessCount: 2,
			Latency:      100,
			RelayCounts:  2,
			Source:       GatewaySource,
			Origin:       "origin",
		}
		// arrange
		metricService.storeAggregatedData(metricData)
		// assertion
		err := checkThatMetricDtoInAggregatedMetricMap(*metricService.AggregatedMetricMap, expectedMetricData)
		if err != nil {
			t.Error(err)
		}
	})

	// Scenario 2 (success relay,Check data that will be added to the previously added data)
	t.Run("SuccessRelay_NonEmptyMap", func(t *testing.T) {
		expectedMetricData = RelayAnalyticsDTO{
			ProjectHash:  "1",
			ChainID:      "testChain",
			APIType:      "testApiType",
			SuccessCount: 3,
			Latency:      150,
			RelayCounts:  3,
			Source:       GatewaySource,
			Origin:       "origin",
		}
		// arrange
		metricService.storeAggregatedData(metricData)
		// assertion
		err := checkThatMetricDtoInAggregatedMetricMap(*metricService.AggregatedMetricMap, expectedMetricData)
		if err != nil {
			t.Error(err)
		}
	})

	// Scenario 3 (failed relay,Check data that will be added to the previously added data)
	t.Run("FailedRelay_NonEmptyMap", func(t *testing.T) {
		metricData.Success = false
		expectedMetricData = RelayAnalyticsDTO{
			ProjectHash:  "1",
			ChainID:      "testChain",
			APIType:      "testApiType",
			SuccessCount: 3,
			Latency:      150,
			RelayCounts:  4,
			Source:       GatewaySource,
			Origin:       "origin",
		}
		// arrange
		metricService.storeAggregatedData(metricData)
		// assertion
		err := checkThatMetricDtoInAggregatedMetricMap(*metricService.AggregatedMetricMap, expectedMetricData)
		if err != nil {
			t.Error(err)
		}
	})

	// Scenario 4 (another project id)
	t.Run("SuccessRelay_WithNewProject_NonEmptyMap", func(t *testing.T) {
		metricData.Success = true
		metricData.ProjectHash = "2"
		expectedMetricData = RelayAnalyticsDTO{
			ProjectHash:  "2",
			ChainID:      "testChain",
			APIType:      "testApiType",
			SuccessCount: 1,
			Latency:      50,
			RelayCounts:  1,
			Source:       GatewaySource,
			Origin:       "origin",
		}
		// arrange
		metricService.storeAggregatedData(metricData)
		// assertion
		err := checkThatMetricDtoInAggregatedMetricMap(*metricService.AggregatedMetricMap, expectedMetricData)
		if err != nil {
			t.Error(err)
		}
	})

	// Scenario 5 (another chain id)
	t.Run("SuccessRelay_WithNewChainId_NonEmptyMap", func(t *testing.T) {
		metricData.Success = true
		metricData.ChainID = "testChain2"
		expectedMetricData = RelayAnalyticsDTO{
			ProjectHash:  "2",
			ChainID:      "testChain2",
			APIType:      "testApiType",
			SuccessCount: 1,
			Latency:      50,
			RelayCounts:  1,
			Source:       GatewaySource,
			Origin:       "origin",
		}
		// arrange
		metricService.storeAggregatedData(metricData)
		// assertion
		err := checkThatMetricDtoInAggregatedMetricMap(*metricService.AggregatedMetricMap, expectedMetricData)
		if err != nil {
			t.Error(err)
		}
	})

	// Scenario 6 (another chain id)
	t.Run("SuccessRelay_WithNewApiType_NonEmptyMap", func(t *testing.T) {
		metricData.Success = true
		metricData.APIType = "testApiType2"
		expectedMetricData = RelayAnalyticsDTO{
			ProjectHash:  "2",
			ChainID:      "testChain2",
			APIType:      "testApiType2",
			SuccessCount: 1,
			Latency:      50,
			RelayCounts:  1,
			Source:       GatewaySource,
			Origin:       "origin",
		}
		// arrange
		metricService.storeAggregatedData(metricData)
		// assertion
		err := checkThatMetricDtoInAggregatedMetricMap(*metricService.AggregatedMetricMap, expectedMetricData)
		if err != nil {
			t.Error(err)
		}
	})
}

func Test_PrepareArrayForProject_OnMetricService(t *testing.T) {
	t.Run("Check_PrepareArrayForProject", func(t *testing.T) {
		// setup
		projectData := map[string]map[string]map[RelaySource]map[string]*AggregatedMetric{
			"testChain": {
				"testApiType": {
					GatewaySource: {
						"origin": {
							TotalLatency: 100,
							RelaysCount:  2,
							SuccessCount: 1,
						},
					},
				},
			},
		}
		expectedMetricData := RelayAnalyticsDTO{
			ProjectHash:  "1",
			ChainID:      "testChain",
			APIType:      "testApiType",
			SuccessCount: 1,
			Latency:      100,
			RelayCounts:  2,
			Source:       GatewaySource,
			Origin:       "origin",
		}

		// arrange
		result := prepareArrayForProject(projectData, expectedMetricData.ProjectHash)

		// assertion
		if len(result) == 0 {
			t.Error("Not enough number of results produced!")
		}
		resultData := result[0]
		if resultData.ProjectHash != expectedMetricData.ProjectHash {
			t.Error("Invalid projectHash on the result array")
		}
		if resultData.ChainID != expectedMetricData.ChainID {
			t.Error("Invalid ChainID on the result array")
		}
		if resultData.APIType != expectedMetricData.APIType {
			t.Error("Invalid APIType on the result array")
		}

		if resultData.Latency != expectedMetricData.Latency {
			t.Error("Invalid Latency on the result array")
		}
		if resultData.SuccessCount != expectedMetricData.SuccessCount {
			t.Error("Invalid Latency on the result array")
		}
		if resultData.RelayCounts != expectedMetricData.RelayCounts {
			t.Error("Invalid Latency on the result array")
		}
	})
}

func checkThatMetricDtoInAggregatedMetricMap(mapData map[string]map[string]map[string]map[RelaySource]map[string]*AggregatedMetric, expectedData RelayAnalyticsDTO) error {
	projectData, projectExists := mapData[expectedData.ProjectHash]
	if !projectExists {
		return fmt.Errorf("Couldn't find project data with key '%s'! ", expectedData.ProjectHash)
	}
	chainIdData, chainIdExists := projectData[expectedData.ChainID]
	if !chainIdExists {
		return fmt.Errorf("Couldn't find chainId data with key '%s'! ", expectedData.ChainID)
	}
	apiTypeData, apiTypeExists := chainIdData[expectedData.APIType]
	if !apiTypeExists {
		return fmt.Errorf("Couldn't find apiType data with key '%s'! ", expectedData.APIType)
	}
	sourceData, sourceExists := apiTypeData[expectedData.Source]
	if !sourceExists {
		return fmt.Errorf("Couldn't find apiType data with key '%d'! ", expectedData.Source)
	}
	data, originExists := sourceData[expectedData.Origin]
	if !originExists {
		return fmt.Errorf("Couldn't find origin data with key '%s'! ", expectedData.Origin)
	}
	if data.RelaysCount != expectedData.RelayCounts {
		return fmt.Errorf("Invalid relayCounts data. expected: '%d' got: '%d'! ", expectedData.RelayCounts, data.RelaysCount)
	}
	if data.TotalLatency != expectedData.Latency {
		return fmt.Errorf("Invalid latency data. expected: '%d' got: '%d'! ", expectedData.Latency, data.TotalLatency)
	}
	if data.SuccessCount != expectedData.SuccessCount {
		return fmt.Errorf("Invalid successCount data. expected: '%d' got: '%d'! ", expectedData.SuccessCount, data.SuccessCount)
	}
	return nil
}
