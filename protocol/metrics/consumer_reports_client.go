package metrics

import (
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
)

const (
	reportName   = "report"
	conflictName = "conflict"
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
	AppendConflict(report ConflictRequest)
}

func NewConflictRequest(request1 *pairingtypes.RelayRequest, result1 *pairingtypes.RelayReply, request2 *pairingtypes.RelayRequest, result2 *pairingtypes.RelayReply) ConflictRequest {
	return ConflictRequest{
		Name: conflictName,
		Conflicts: []ConflictContainer{{
			Request: *request1,
			Reply:   *result1,
		}, {
			Request: *request2,
			Reply:   *result2,
		}},
	}
}

type ConflictContainer struct {
	Request pairingtypes.RelayRequest `json:"request"`
	Reply   pairingtypes.RelayReply   `json:"reply"`
}
type ConflictRequest struct {
	Name      string              `json:"name"`
	Conflicts []ConflictContainer `json:"conflicts"`
}

func (rr ConflictRequest) String() string {
	rr.Name = conflictName
	bytes, err := json.Marshal(rr)
	if err != nil {
		return ""
	}
	return string(bytes)
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

func (cuc *ConsumerReportsClient) AppendConflict(report ConflictRequest) {
	if cuc == nil {
		return
	}
	cuc.appendQueue(report)
}
