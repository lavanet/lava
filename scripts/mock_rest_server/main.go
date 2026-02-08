package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

type Config struct {
	mu       sync.RWMutex
	response string
	status   int
}

var config = &Config{
	status: 200,
	response: `{"tx_response":{"height":"0","txhash":"B303F5540A6CDDD8CEECD3F7CEF1F3913440E9047E0403EE5614C15B177687F6","codespace":"sdk","code":32,"data":"","raw_log":"account sequence mismatch, expected 8, got 4: incorrect account sequence","logs":[],"info":"","gas_wanted":"0","gas_used":"0","tx":null,"timestamp":"","events":[]}}`,
}

// Predefined responses for validation endpoints (allows startup to succeed)
var validationResponses = map[string]string{
	// Latest block - used for startup validation and block tracking
	"/cosmos/base/tendermint/v1beta1/blocks/latest": `{"block_id":{"hash":"MOCK_HASH","part_set_header":{"total":1,"hash":"MOCK_PARTS"}},"block":{"header":{"chain_id":"lava-testnet-2","height":"1000000","time":"2024-01-01T00:00:00Z"}}}`,
	// Node info - used for chain-id verification
	"/cosmos/base/tendermint/v1beta1/node_info": `{"default_node_info":{"network":"lava-testnet-2"},"application_version":{"name":"lava","version":"1.0.0"}}`,
	// Syncing status
	"/cosmos/base/tendermint/v1beta1/syncing": `{"syncing":false}`,
}

// Handler for all requests
func requestHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	// Check if this is a validation endpoint that needs a proper response
	if validationResponse, isValidation := validationResponses[path]; isValidation {
		log.Printf("📋 Validation Request: %s %s from %s -> returning mock validation response", r.Method, path, r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(validationResponse))
		return
	}

	// Also handle block-by-height requests for validation
	if strings.HasPrefix(path, "/cosmos/base/tendermint/v1beta1/blocks/") {
		blockNum := strings.TrimPrefix(path, "/cosmos/base/tendermint/v1beta1/blocks/")
		log.Printf("📋 Block Request: %s %s (block=%s) from %s -> returning mock block", r.Method, path, blockNum, r.RemoteAddr)
		response := fmt.Sprintf(`{"block_id":{"hash":"MOCK_HASH_%s"},"block":{"header":{"chain_id":"lava-testnet-2","height":"%s","time":"2024-01-01T00:00:00Z"}}}`, blockNum, blockNum)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(response))
		return
	}

	// For all other requests (like /cosmos/tx/v1beta1/txs), use configured response
	config.mu.RLock()
	status := config.status
	response := config.response
	config.mu.RUnlock()

	log.Printf("⚠️  API Request: %s %s from %s -> returning configured response (node error)", r.Method, path, r.RemoteAddr)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(response))
	
	log.Printf("Response: status=%d", status)
}

// Control endpoint to change response
func controlHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	
	config.mu.Lock()
	
	if statusStr := query.Get("status"); statusStr != "" {
		var status int
		fmt.Sscanf(statusStr, "%d", &status)
		if status > 0 {
			config.status = status
			log.Printf("Control: status set to %d", status)
		}
	}
	
	if response := query.Get("response"); response != "" {
		config.response = response
		log.Printf("Control: response set to: %s", response)
	}
	
	if responseType := query.Get("type"); responseType != "" {
		switch responseType {
		case "sequence_error":
			config.response = `{"tx_response":{"height":"0","txhash":"B303F5540A6CDDD8CEECD3F7CEF1F3913440E9047E0403EE5614C15B177687F6","codespace":"sdk","code":32,"data":"","raw_log":"account sequence mismatch, expected 8, got 4: incorrect account sequence","logs":[],"info":"","gas_wanted":"0","gas_used":"0","tx":null,"timestamp":"","events":[]}}`
			log.Println("Control: response set to sequence_error")
		case "success":
			config.response = `{"tx_response":{"height":"0","txhash":"MOCK_SUCCESS_HASH","codespace":"","code":0,"data":"","raw_log":"[]","logs":[],"info":"","gas_wanted":"0","gas_used":"0","tx":null,"timestamp":"","events":[]}}`
			log.Println("Control: response set to success")
		case "insufficient_fee":
			config.response = `{"tx_response":{"height":"0","txhash":"MOCK_FEE_ERROR_HASH","codespace":"sdk","code":13,"data":"","raw_log":"insufficient fees; got:  required: 1ulava: insufficient fee","logs":[],"info":"","gas_wanted":"0","gas_used":"0","tx":null,"timestamp":"","events":[]}}`
			log.Println("Control: response set to insufficient_fee")
		}
	}
	
	currentConfig := map[string]interface{}{
		"status":   config.status,
		"response": config.response,
	}
	config.mu.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentConfig)
}

func main() {
	port := flag.Int("port", 9999, "Port to listen on")
	flag.Parse()

	http.HandleFunc("/control", controlHandler)
	http.HandleFunc("/", requestHandler)

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Printf("🚀 Mock REST server starting on http://%s", addr)
	log.Printf("📋 Default response: Node error (sequence mismatch)")
	log.Printf("🔧 Control endpoint: http://localhost:%d/control", *port)
	log.Printf("\nExamples:")
	log.Printf("  - Change to success: curl 'http://localhost:%d/control?type=success'", *port)
	log.Printf("  - Change to error:   curl 'http://localhost:%d/control?type=sequence_error'", *port)
	log.Printf("  - HTTP 500:          curl 'http://localhost:%d/control?status=500'", *port)
	log.Printf("  - Reset to 200:      curl 'http://localhost:%d/control?status=200'", *port)
	
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
