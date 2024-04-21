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
		serverClient.AppendReferrer(NewReferrerRequest("banana", "ETH1", "Message-1", "https://referer.com", "https://origin.com", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0"))
		serverClient.AppendReferrer(NewReferrerRequest("banana", "COSMOSHUB", "Message-2", "https://referer.com", "https://origin.com", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0"))
		serverClient.AppendReferrer(NewReferrerRequest("papaya", "ETH1", "Message-3", "https://referer.com", "https://origin.com", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0"))
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
		serverClient.AppendReferrer(NewReferrerRequest("banana", "ETH1", "Message-1", "https://referer.com", "https://origin.com", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0"))
		time.Sleep(110 * time.Millisecond)
		getSender := func() ReferrerSender {
			return serverClient
		}
		reporter := getSender()
		reporter.AppendReferrer(NewReferrerRequest("banana", "ETH1", "Message-2", "https://referer.com", "https://origin.com", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0"))
	})
}
