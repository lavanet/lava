package metrics

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v5/utils"
)

type QueueSender struct {
	name                string
	endpointAddress     string
	addQueue            []fmt.Stringer
	ticker              *time.Ticker
	lock                sync.RWMutex
	sendID              int
	isSendQueueRunning  bool
	aggregationFunction func([]fmt.Stringer) []fmt.Stringer
}

func NewQueueSender(endpointAddress string, name string, aggregationFunction func([]fmt.Stringer) []fmt.Stringer, interval ...time.Duration) *QueueSender {
	if endpointAddress == "" {
		return nil
	}
	tickerTime := 30 * time.Second
	if len(interval) > 0 {
		tickerTime = interval[0]
	}
	cuc := &QueueSender{
		name:                name,
		endpointAddress:     endpointAddress,
		ticker:              time.NewTicker(tickerTime),
		addQueue:            make([]fmt.Stringer, 0),
		aggregationFunction: aggregationFunction,
	}

	go cuc.sendQueueStart()

	return cuc
}

func (cuc *QueueSender) sendQueueStart() {
	if cuc == nil {
		return
	}
	utils.LavaFormatDebug("[QueueSender] Starting sendQueueStart loop", utils.LogAttr("name", cuc.name))
	for range cuc.ticker.C {
		cuc.sendQueueTick()
	}
}

func (crc *QueueSender) sendQueueTick() {
	if crc == nil {
		return
	}

	crc.lock.Lock()
	defer crc.lock.Unlock()

	if len(crc.addQueue) == 0 {
		utils.LavaFormatDebug("[QueueSender] sendQueueTick: addQueue is empty", utils.LogAttr("name", crc.name))
		return
	}

	if !crc.isSendQueueRunning {
		sendQueue := crc.addQueue
		crc.addQueue = make([]fmt.Stringer, 0)
		crc.isSendQueueRunning = true
		crc.sendID++
		utils.LavaFormatDebug("[QueueSender] Swapped queues", utils.LogAttr("name", crc.name), utils.LogAttr("sendQueue_length", len(sendQueue)), utils.LogAttr("send_id", crc.sendID))

		sendID := crc.sendID
		cucEndpointAddress := crc.endpointAddress

		go func() {
			crc.sendData(sendQueue, sendID, cucEndpointAddress)

			crc.lock.Lock()
			crc.isSendQueueRunning = false
			crc.lock.Unlock()
		}()
	} else {
		utils.LavaFormatDebug("[QueueSender] server is busy skipping send", utils.LogAttr("name", crc.name), utils.LogAttr("id", crc.sendID))
	}
}

func (cuc *QueueSender) appendQueue(request fmt.Stringer) {
	if cuc == nil {
		return
	}
	cuc.lock.Lock()
	defer cuc.lock.Unlock()
	cuc.addQueue = append(cuc.addQueue, request)
}

func (crc *QueueSender) send(sendQueue []fmt.Stringer, sendID int, endpointAddress string) (*http.Response, error) {
	if crc == nil {
		return nil, utils.LavaFormatError("QueueSender is nil. misuse detected", nil)
	}

	if len(sendQueue) == 0 {
		return nil, errors.New("sendQueue is empty")
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	jsonData, err := json.Marshal(sendQueue)
	if err != nil {
		return nil, utils.LavaFormatError("Failed marshaling aggregated requests", err)
	}

	var resp *http.Response
	for i := 0; i < 3; i++ {
		resp, err = client.Post(endpointAddress, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			utils.LavaFormatDebug("[QueueSender] Failed to post request", utils.LogAttr("name", crc.name), utils.LogAttr("Attempt", i+1), utils.LogAttr("err", err))
			time.Sleep(2 * time.Second)
		} else {
			return resp, nil
		}
	}

	return nil, utils.LavaFormatWarning("[QueueSender] Failed to send requests after 3 attempts", err, utils.LogAttr("name", crc.name))
}

func (crc *QueueSender) handleSendResponse(resp *http.Response, sendID int) {
	if crc == nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			utils.LavaFormatWarning("[QueueSender] failed reading response body", err, utils.LogAttr("name", crc.name))
		} else {
			utils.LavaFormatWarning("[QueueSender] Received non-200 status code", nil, utils.LogAttr("name", crc.name), utils.LogAttr("status_code", resp.StatusCode), utils.LogAttr("body", string(bodyBytes)))
		}
	}
}

func (cuc *QueueSender) sendData(sendQueue []fmt.Stringer, sendID int, cucEndpointAddress string) {
	if cuc == nil {
		return
	}
	if cuc.aggregationFunction != nil {
		sendQueue = cuc.aggregationFunction(sendQueue)
	}
	resp, err := cuc.send(sendQueue, sendID, cucEndpointAddress)
	if err != nil {
		utils.LavaFormatWarning("[QueueSender] failed sendRelay data", err)
		return
	}
	cuc.handleSendResponse(resp, sendID)
}
