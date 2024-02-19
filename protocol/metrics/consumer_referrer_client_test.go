package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReferrerClientFlows(t *testing.T) {
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
		serverClient := NewConsumerReferrerClient(endpoint, 100*time.Millisecond)
		serverClient.AppendReferrer(NewReferrerRequest("banana"))
		serverClient.AppendReferrer(NewReferrerRequest("banana"))
		serverClient.AppendReferrer(NewReferrerRequest("papaya"))
		time.Sleep(110 * time.Millisecond)
		require.Len(t, messages, 2)
		bananas := 0
		papayas := 0
		for _, message := range messages {
			if message["referer-id"] == "banana" {
				bananas++
				require.Equal(t, message["count"], 2.0)
			} else if message["referer-id"] == "papaya" {
				papayas++
			}
		}
		require.Equal(t, bananas, 1)
		require.Equal(t, papayas, 1)
	})
}

func TestReferrerClientNull(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		serverClient := NewConsumerReferrerClient("")
		require.Nil(t, serverClient)
		serverClient.AppendReferrer(NewReferrerRequest("banana"))
		time.Sleep(110 * time.Millisecond)
		getSender := func() ReferrerSender {
			return serverClient
		}
		reporter := getSender()
		reporter.AppendReferrer(NewReferrerRequest("banana"))
	})
}
