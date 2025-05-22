package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/utils"
	"github.com/stretchr/testify/require"
)

func TestReportsClientFlows(t *testing.T) {
	t.Run("one-shot", func(t *testing.T) {
		serverWaitGroup := utils.NewChanneledWaitGroup()
		serverWaitGroup.Add(2) // 2 reports
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
			for range messages {
				serverWaitGroup.Done()
			}
		})

		mockServer := httptest.NewServer(serverHandle)
		defer mockServer.Close()
		endpoint := mockServer.URL
		serverClient := NewConsumerReportsClient(endpoint, 100*time.Millisecond)
		serverClient.AppendReport(NewReportsRequest("lava@test", []error{fmt.Errorf("bad"), fmt.Errorf("very-bad")}, "LAV1"))
		serverClient.AppendReport(NewReportsRequest("lava@test", []error{fmt.Errorf("bad"), fmt.Errorf("very-bad")}, "LAV1"))

		select {
		case <-serverWaitGroup.Wait():
			// all done
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout reached before reports were received")
		}

		require.Len(t, messages, 3)
		reports := 0
		for _, message := range messages {
			if message["name"] == "report" {
				reports++
			}
		}
		require.Equal(t, reports, 2)
	})
}
