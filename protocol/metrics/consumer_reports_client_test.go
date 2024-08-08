package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestReportsClientFlows(t *testing.T) {
	t.Run("one-shot", func(t *testing.T) {
		messages := []map[string]interface{}{}
		reqMap := []map[string]interface{}{}
		serverHandle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle the incoming request and provide the desired response
			data := make([]byte, r.ContentLength)
			r.Body.Read(data)
			err := json.Unmarshal(data, &reqMap)
			require.NoError(t, err)
			messages = append(messages, reqMap...)
			reqMap = []map[string]interface{}{}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x10a7a08"}`)
		})

		mockServer := httptest.NewServer(serverHandle)
		defer mockServer.Close()
		endpoint := mockServer.URL
		serverClient := NewConsumerReportsClient(endpoint, 100*time.Millisecond)
		serverClient.AppendReport(NewReportsRequest("lava@test", []error{fmt.Errorf("bad"), fmt.Errorf("very-bad")}, "LAV1"))
		serverClient.AppendReport(NewReportsRequest("lava@test", []error{fmt.Errorf("bad"), fmt.Errorf("very-bad")}, "LAV1"))
		serverClient.AppendConflict(NewConflictRequest(&pairingtypes.RelayRequest{
			RelaySession: &pairingtypes.RelaySession{Provider: "lava@conflict0"},
			RelayData:    &pairingtypes.RelayPrivateData{},
		}, &pairingtypes.RelayReply{
			Data:                  []byte{1, 2, 3},
			Sig:                   []byte{},
			LatestBlock:           0,
			FinalizedBlocksHashes: []byte{},
			SigBlocks:             []byte{},
			Metadata:              []pairingtypes.Metadata{},
		}, &pairingtypes.RelayRequest{}, &pairingtypes.RelayReply{}))
		time.Sleep(110 * time.Millisecond)
		require.Len(t, messages, 3)
		reports := 0
		conflicts := 0
		for _, message := range messages {
			if message["name"] == "report" {
				reports++
			} else if message["name"] == "conflict" {
				conflicts++
			}
		}
		require.Equal(t, reports, 2)
		require.Equal(t, conflicts, 1)
	})
}

func TestReportsClientNull(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		serverClient := NewConsumerReportsClient("")
		require.Nil(t, serverClient)
		serverClient.AppendReport(NewReportsRequest("lava@test", []error{fmt.Errorf("bad"), fmt.Errorf("very-bad")}, "LAV1"))
		conflictData := NewConflictRequest(&pairingtypes.RelayRequest{
			RelaySession: &pairingtypes.RelaySession{Provider: "lava@conflict0"},
			RelayData:    &pairingtypes.RelayPrivateData{},
		}, &pairingtypes.RelayReply{
			Data:                  []byte{1, 2, 3},
			Sig:                   []byte{},
			LatestBlock:           0,
			FinalizedBlocksHashes: []byte{},
			SigBlocks:             []byte{},
			Metadata:              []pairingtypes.Metadata{},
		}, &pairingtypes.RelayRequest{}, &pairingtypes.RelayReply{})
		serverClient.AppendConflict(conflictData)
		time.Sleep(110 * time.Millisecond)
		getReporter := func() Reporter {
			return serverClient
		}
		reporter := getReporter()
		reporter.AppendConflict(conflictData)
		reporter.AppendReport(NewReportsRequest("lava@test", []error{fmt.Errorf("bad"), fmt.Errorf("very-bad")}, "LAV1"))
	})
}
