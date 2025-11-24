package cache_populator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

const (
	FlagLogLevel = "log_level"
)

// Request represents a single HTTP request configuration
type Request struct {
	Type    string            `json:"type,omitempty"`
	Body    string            `json:"body,omitempty"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// CacheConfig represents a single server configuration with its requests
type CacheConfig struct {
	ServerAddress  string        `json:"server_address"`
	Interval       time.Duration `json:"interval"`
	ContextTimeout time.Duration `json:"context_timeout"`
	Requests       []Request     `json:"requests"`
}

// UnmarshalJSON implements custom JSON unmarshaling for CacheConfig
func (c *CacheConfig) UnmarshalJSON(data []byte) error {
	type Alias CacheConfig
	aux := &struct {
		Interval       string `json:"interval"`
		ContextTimeout string `json:"context_timeout"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	duration, err := time.ParseDuration(aux.Interval)
	if err != nil {
		return fmt.Errorf("invalid interval format: %w", err)
	}
	c.Interval = duration

	duration, err = time.ParseDuration(aux.ContextTimeout)
	if err != nil {
		return fmt.Errorf("invalid context timeout format: %w", err)
	}
	c.ContextTimeout = duration
	return nil
}

// executeRequest handles a single request execution with its own ticker
func executeRequest(config CacheConfig, request Request) {
	client := common.OptimizedHttpClient()
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()
	for range ticker.C {
		go func() {
			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), config.ContextTimeout)

			// Generate new GUID for this request cycle
			guid := utils.GenerateUniqueIdentifier()

			// Create request based on type
			var httpReq *http.Request
			var err error

			switch request.Type {
			case "GET":
				httpReq, err = http.NewRequestWithContext(ctx, http.MethodGet,
					config.ServerAddress+request.Path, nil)
			case "POST":
				httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost,
					config.ServerAddress+request.Path, bytes.NewBufferString(request.Body))
			default:
				utils.LavaFormatError("unsupported request type", fmt.Errorf("type: %s", request.Type), utils.Attribute{Key: "guid", Value: guid})
				cancel()
				return
			}

			if err != nil {
				utils.LavaFormatError("failed to create request", err, utils.Attribute{Key: "guid", Value: guid})
				cancel()
				return
			}

			// Add headers to the request
			for key, value := range request.Headers {
				httpReq.Header.Set(key, value)
			}

			// Execute request
			utils.LavaFormatDebug("request ", utils.Attribute{Key: "path", Value: request.Path}, utils.Attribute{Key: "body", Value: request.Body}, utils.Attribute{Key: "guid", Value: guid})

			resp, err := client.Do(httpReq)
			if err != nil {
				utils.LavaFormatError("failed to execute request", err, utils.Attribute{Key: "guid", Value: guid})
				cancel()
				// Wait for next tick before retrying
				return
			}

			if resp.StatusCode >= 400 {
				utils.LavaFormatError("request failed",
					fmt.Errorf("status code: %d, path: %s", resp.StatusCode, request.Path), utils.Attribute{Key: "guid", Value: guid})
			}

			resp.Body.Close()
			utils.LavaFormatDebug("response", utils.Attribute{Key: "status_code", Value: resp.StatusCode}, utils.Attribute{Key: "path", Value: request.Path}, utils.Attribute{Key: "guid", Value: guid}, utils.Attribute{Key: "body", Value: request.Body})
			cancel()
		}()
	}
}

func CreateCachePopulatorCommand() *cobra.Command {
	cacheCmd := &cobra.Command{
		Use:   "cache-populator [json-file]",
		Short: "populate a cache server with entries from a JSON file",
		Long: `populate a cache server with entries from a JSON file. The JSON file should contain an array of cache entries 
with keys and values to be populated into the cache server at the specified address.`,
		Example: `cache-populator entries.json`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonFilePath := args[0]

			// ctx := context.Background()
			logLevel, err := cmd.Flags().GetString(FlagLogLevel)
			if err != nil {
				utils.LavaFormatFatal("failed to read log level flag", err)
			}
			utils.SetGlobalLoggingLevel(logLevel)

			// Read and parse the JSON file
			jsonData, err := os.ReadFile(jsonFilePath)
			if err != nil {
				return fmt.Errorf("failed to read JSON file: %w", err)
			}

			var configs []CacheConfig
			if err := json.Unmarshal(jsonData, &configs); err != nil {
				return fmt.Errorf("failed to parse JSON configuration: %w", err)
			}

			for _, config := range configs {
				fmt.Printf("Starting Populator with the following configuration: %s\n", config.ServerAddress)
				fmt.Printf("Server Address: %s\n", config.ServerAddress)
				fmt.Printf("Interval: %s\n", config.Interval)
				fmt.Printf("Number of Requests: %d\n", len(config.Requests))
				fmt.Printf("Context Timeout: %s\n", config.ContextTimeout)

				// Launch each request in its own goroutine
				for _, request := range config.Requests {
					go executeRequest(config, request)
				}
			}

			// Block forever since requests run indefinitely
			select {}
		},
	}
	cacheCmd.Flags().String(FlagLogLevel, zerolog.InfoLevel.String(), "The logging level (trace|debug|info|warn|error|fatal|panic)")
	return cacheCmd
}
