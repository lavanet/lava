package metrics

import (
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v5/utils"
)

const (
	reportName = "report"
)

type ConsumerReportsClient struct {
	*QueueSender
}

func NewReportsRequest(provider string, errors []error, specId string) ReportsRequest {
	errorsStrings := []string{}
	for _, err := range errors {
		if err == nil {
			continue
		}
		errorsStrings = append(errorsStrings, err.Error())
	}
	return ReportsRequest{
		Name:     reportName,
		Errors:   strings.Join(errorsStrings, ","),
		Provider: provider,
		SpecId:   specId,
	}
}

type ReportsRequest struct {
	Name     string `json:"name"`
	Errors   string `json:"errors"`
	Provider string `json:"provider"`
	SpecId   string `json:"spec_id"`
}

func (rr ReportsRequest) String() string {
	rr.Name = reportName
	bytes, err := json.Marshal(rr)
	if err != nil {
		return ""
	}
	return string(bytes)
}

type Reporter interface {
	AppendReport(report ReportsRequest)
}

func NewConsumerReportsClient(endpointAddress string, interval ...time.Duration) *ConsumerReportsClient {
	if endpointAddress == "" {
		utils.LavaFormatInfo("Running with Consumer Reports Client Disabled")
		return nil
	}

	cuc := &ConsumerReportsClient{
		QueueSender: NewQueueSender(endpointAddress, "ConsumerReports", nil, interval...),
	}
	return cuc
}

func (cuc *ConsumerReportsClient) AppendReport(report ReportsRequest) {
	if cuc == nil {
		return
	}
	cuc.appendQueue(report)
}
