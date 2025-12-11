package streamer

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lavanet/lava/v5/utils"
)

// Common event signatures
const (
	// ERC20 Events
	EventSignatureTransfer = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" // Transfer(address,address,uint256)
	EventSignatureApproval = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925" // Approval(address,address,uint256)

	// ERC721 Events
	EventSignatureTransferNFT    = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" // Transfer(address,address,uint256) - same as ERC20
	EventSignatureApprovalNFT    = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925" // Approval(address,address,uint256) - same as ERC20
	EventSignatureApprovalForAll = "0x17307eab39ab6107e8899845ad3d59bd9653f200f220920489ca2b5937696c31" // ApprovalForAll(address,address,bool)

	// Common DeFi Events
	EventSignatureSwap = "0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822" // Swap(address,uint256,uint256,uint256,uint256,address)
	EventSignatureMint = "0x4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f" // Mint(address,uint256,uint256)
	EventSignatureBurn = "0xdccd412f0b1252819cb1fd330b93224ca42612892bb3f4f789976e6d81936496" // Burn(address,uint256,uint256,address)
)

// EventType represents the type of event
type EventType string

const (
	EventTypeUnknown        EventType = "unknown"
	EventTypeTransfer       EventType = "transfer"
	EventTypeApproval       EventType = "approval"
	EventTypeApprovalForAll EventType = "approval_for_all"
	EventTypeSwap           EventType = "swap"
	EventTypeMint           EventType = "mint"
	EventTypeBurn           EventType = "burn"
	EventTypeCustom         EventType = "custom"
)

// DecodedEvent represents a decoded event with its parameters
type DecodedEvent struct {
	EventType  EventType              `json:"event_type"`
	EventName  string                 `json:"event_name"`
	Signature  string                 `json:"signature"`
	Parameters map[string]interface{} `json:"parameters"`
	RawLog     *Log                   `json:"raw_log,omitempty"`
}

// ABIDecoder handles ABI-based event decoding
type ABIDecoder struct {
	// Contract ABIs by address
	abis map[string]*abi.ABI

	// Event signatures to ABI mappings
	eventSignatures map[string]*abi.Event

	// Standard ERC20/ERC721 ABIs
	erc20ABI  *abi.ABI
	erc721ABI *abi.ABI
}

// NewABIDecoder creates a new ABI decoder
func NewABIDecoder() *ABIDecoder {
	decoder := &ABIDecoder{
		abis:            make(map[string]*abi.ABI),
		eventSignatures: make(map[string]*abi.Event),
	}

	// Load standard ABIs
	decoder.loadStandardABIs()

	return decoder
}

// loadStandardABIs loads common ERC20 and ERC721 ABIs
func (d *ABIDecoder) loadStandardABIs() {
	// ERC20 ABI (minimal for events)
	erc20JSON := `[
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "name": "from", "type": "address"},
				{"indexed": true, "name": "to", "type": "address"},
				{"indexed": false, "name": "value", "type": "uint256"}
			],
			"name": "Transfer",
			"type": "event"
		},
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "name": "owner", "type": "address"},
				{"indexed": true, "name": "spender", "type": "address"},
				{"indexed": false, "name": "value", "type": "uint256"}
			],
			"name": "Approval",
			"type": "event"
		}
	]`

	// ERC721 ABI (minimal for events)
	erc721JSON := `[
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "name": "from", "type": "address"},
				{"indexed": true, "name": "to", "type": "address"},
				{"indexed": true, "name": "tokenId", "type": "uint256"}
			],
			"name": "Transfer",
			"type": "event"
		},
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "name": "owner", "type": "address"},
				{"indexed": true, "name": "approved", "type": "address"},
				{"indexed": true, "name": "tokenId", "type": "uint256"}
			],
			"name": "Approval",
			"type": "event"
		},
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "name": "owner", "type": "address"},
				{"indexed": true, "name": "operator", "type": "address"},
				{"indexed": false, "name": "approved", "type": "bool"}
			],
			"name": "ApprovalForAll",
			"type": "event"
		}
	]`

	var err error
	d.erc20ABI, err = abi.JSON(strings.NewReader(erc20JSON))
	if err != nil {
		utils.LavaFormatWarning("Failed to parse ERC20 ABI", err)
	}

	d.erc721ABI, err = abi.JSON(strings.NewReader(erc721JSON))
	if err != nil {
		utils.LavaFormatWarning("Failed to parse ERC721 ABI", err)
	}

	// Register standard event signatures
	if d.erc20ABI != nil {
		for _, event := range d.erc20ABI.Events {
			sig := "0x" + hex.EncodeToString(event.ID.Bytes())
			d.eventSignatures[sig] = &event
		}
	}

	if d.erc721ABI != nil {
		for _, event := range d.erc721ABI.Events {
			sig := "0x" + hex.EncodeToString(event.ID.Bytes())
			// Only add if not already present (avoid ERC20/ERC721 overlap)
			if _, exists := d.eventSignatures[sig]; !exists {
				d.eventSignatures[sig] = &event
			}
		}
	}
}

// RegisterABI registers a contract ABI for decoding
func (d *ABIDecoder) RegisterABI(contractAddress string, abiJSON string) error {
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Normalize address
	addr := strings.ToLower(contractAddress)
	d.abis[addr] = &parsedABI

	// Register event signatures
	for _, event := range parsedABI.Events {
		sig := "0x" + hex.EncodeToString(event.ID.Bytes())
		d.eventSignatures[sig] = &event
	}

	utils.LavaFormatInfo("Registered ABI for contract",
		utils.LogAttr("address", contractAddress),
		utils.LogAttr("events", len(parsedABI.Events)),
	)

	return nil
}

// DecodeLog decodes a log entry using registered ABIs
func (d *ABIDecoder) DecodeLog(log *Log) (*DecodedEvent, error) {
	if log.Topic0 == nil {
		return nil, fmt.Errorf("log has no topic0")
	}

	// Get event signature
	signature := *log.Topic0

	// Try to find matching event
	event, found := d.eventSignatures[signature]
	if !found {
		// Try contract-specific ABI
		addr := strings.ToLower(log.Address)
		if contractABI, exists := d.abis[addr]; exists {
			for _, evt := range contractABI.Events {
				sig := "0x" + hex.EncodeToString(evt.ID.Bytes())
				if sig == signature {
					event = &evt
					found = true
					break
				}
			}
		}
	}

	if !found {
		return &DecodedEvent{
			EventType:  EventTypeUnknown,
			EventName:  "Unknown",
			Signature:  signature,
			Parameters: make(map[string]interface{}),
			RawLog:     log,
		}, nil
	}

	// Decode event
	decoded, err := d.decodeEvent(event, log)
	if err != nil {
		return nil, fmt.Errorf("failed to decode event: %w", err)
	}

	decoded.RawLog = log
	return decoded, nil
}

// decodeEvent decodes an event using its ABI definition
func (d *ABIDecoder) decodeEvent(event *abi.Event, log *Log) (*DecodedEvent, error) {
	// Collect indexed and non-indexed parameters
	var indexed []interface{}
	var nonIndexed []interface{}

	// Parse topics (indexed parameters)
	topics := []string{}
	if log.Topic0 != nil {
		topics = append(topics, *log.Topic0)
	}
	if log.Topic1 != nil {
		topics = append(topics, *log.Topic1)
	}
	if log.Topic2 != nil {
		topics = append(topics, *log.Topic2)
	}
	if log.Topic3 != nil {
		topics = append(topics, *log.Topic3)
	}

	// Skip topic0 (event signature)
	topicIndex := 1
	for _, input := range event.Inputs {
		if input.Indexed {
			if topicIndex < len(topics) {
				value := d.decodeParameter(input.Type, topics[topicIndex])
				indexed = append(indexed, value)
				topicIndex++
			}
		}
	}

	// Parse data (non-indexed parameters)
	if log.Data != "" && log.Data != "0x" {
		dataBytes, err := hexToBytes(log.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode data: %w", err)
		}

		// Get non-indexed arguments
		var nonIndexedArgs abi.Arguments
		for _, input := range event.Inputs {
			if !input.Indexed {
				nonIndexedArgs = append(nonIndexedArgs, input)
			}
		}

		if len(nonIndexedArgs) > 0 {
			values, err := nonIndexedArgs.Unpack(dataBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack data: %w", err)
			}
			nonIndexed = values
		}
	}

	// Build parameters map
	parameters := make(map[string]interface{})
	indexedIdx := 0
	nonIndexedIdx := 0

	for _, input := range event.Inputs {
		var value interface{}
		if input.Indexed {
			if indexedIdx < len(indexed) {
				value = indexed[indexedIdx]
				indexedIdx++
			}
		} else {
			if nonIndexedIdx < len(nonIndexed) {
				value = nonIndexed[nonIndexedIdx]
				nonIndexedIdx++
			}
		}

		if value != nil {
			parameters[input.Name] = d.formatValue(value)
		}
	}

	// Determine event type
	eventType := d.getEventType(event.Name, *log.Topic0)

	return &DecodedEvent{
		EventType:  eventType,
		EventName:  event.Name,
		Signature:  *log.Topic0,
		Parameters: parameters,
	}, nil
}

// decodeParameter decodes a single parameter from a topic
func (d *ABIDecoder) decodeParameter(typ abi.Type, topic string) interface{} {
	bytes, err := hexToBytes(topic)
	if err != nil {
		return topic
	}

	switch typ.T {
	case abi.AddressTy:
		return common.BytesToAddress(bytes).Hex()
	case abi.UintTy, abi.IntTy:
		return new(big.Int).SetBytes(bytes)
	case abi.BoolTy:
		return bytes[len(bytes)-1] == 1
	case abi.BytesTy, abi.FixedBytesTy:
		return "0x" + hex.EncodeToString(bytes)
	default:
		return topic
	}
}

// formatValue formats a decoded value for JSON serialization
func (d *ABIDecoder) formatValue(value interface{}) interface{} {
	switch v := value.(type) {
	case *big.Int:
		return v.String()
	case common.Address:
		return v.Hex()
	case []byte:
		return "0x" + hex.EncodeToString(v)
	case [32]byte:
		return "0x" + hex.EncodeToString(v[:])
	default:
		return value
	}
}

// getEventType determines the type of event based on name and signature
func (d *ABIDecoder) getEventType(name string, signature string) EventType {
	switch signature {
	case EventSignatureTransfer:
		return EventTypeTransfer
	case EventSignatureApproval:
		return EventTypeApproval
	case EventSignatureApprovalForAll:
		return EventTypeApprovalForAll
	case EventSignatureSwap:
		return EventTypeSwap
	case EventSignatureMint:
		return EventTypeMint
	case EventSignatureBurn:
		return EventTypeBurn
	default:
		return EventTypeCustom
	}
}

// Helper function to convert hex string to bytes
func hexToBytes(hexStr string) ([]byte, error) {
	if strings.HasPrefix(hexStr, "0x") {
		hexStr = hexStr[2:]
	}
	return hex.DecodeString(hexStr)
}

// GetEventTypeInfo returns human-readable information about an event type
func GetEventTypeInfo(eventType EventType) string {
	switch eventType {
	case EventTypeTransfer:
		return "Token or NFT Transfer"
	case EventTypeApproval:
		return "Token or NFT Approval"
	case EventTypeApprovalForAll:
		return "NFT Approval For All"
	case EventTypeSwap:
		return "Token Swap"
	case EventTypeMint:
		return "Token Mint"
	case EventTypeBurn:
		return "Token Burn"
	case EventTypeCustom:
		return "Custom Event"
	default:
		return "Unknown Event"
	}
}
