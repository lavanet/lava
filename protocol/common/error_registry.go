package common

import (
	"fmt"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Core types
// ---------------------------------------------------------------------------

// ErrorCategory — top-level grouping: internal (Lava-introduced) vs external (pass-through)
type ErrorCategory int

const (
	CategoryInternal ErrorCategory = iota // Errors introduced by Lava — user would never see these without Lava
	CategoryExternal                      // Pass-through errors — user would get the same error talking to the node directly
)

func (c ErrorCategory) String() string {
	switch c {
	case CategoryInternal:
		return "internal"
	case CategoryExternal:
		return "external"
	default:
		return "unknown"
	}
}

// ErrorSubCategory — finer classification within each category.
// Subcategories carry behavioral implications (e.g., UnsupportedMethod triggers zero retries, zero CU, caching).
type ErrorSubCategory int

const (
	SubCategoryNone              ErrorSubCategory = iota
	SubCategoryUnsupportedMethod                  // zero retries, zero CU, cached response, no provider scoring
)

func (sc ErrorSubCategory) String() string {
	switch sc {
	case SubCategoryUnsupportedMethod:
		return "unsupported_method"
	default:
		return "none"
	}
}

func (sc ErrorSubCategory) IsUnsupportedMethod() bool {
	return sc == SubCategoryUnsupportedMethod
}

// LavaError is the central error definition — a classification struct, not a Go error.
// It is metadata *about* an error, used for logging, metrics, and retry decisions.
// The original error always passes through unchanged to the user (transparent hop).
type LavaError struct {
	Code        uint32
	Name        string
	Category    ErrorCategory
	SubCategory ErrorSubCategory
	Description string
	Retryable   bool // retrying same relay with same params to a different provider has a chance of succeeding
}

func (le *LavaError) String() string {
	return fmt.Sprintf("[%d] %s", le.Code, le.Name)
}

// Error implements the error interface so LavaError can be used directly as an error
// and checked with errors.Is().
func (le *LavaError) Error() string {
	return fmt.Sprintf("%s: %s", le.Name, le.Description)
}

// Is implements errors.Is matching by error code.
// This allows errors.Is(err, LavaErrorSomething) to work when err wraps a LavaError.
func (le *LavaError) Is(target error) bool {
	if t, ok := target.(*LavaError); ok {
		return le.Code == t.Code
	}
	return false
}

// ABCICode returns the error code for gRPC wire protocol compatibility.
// This replaces the sdkerrors.ABCICode() method so LavaError can be used
// in status.Error(codes.Code(err.ABCICode()), ...) calls.
func (le *LavaError) ABCICode() uint32 {
	return le.Code
}

// ---------------------------------------------------------------------------
// Chain families — grouping chains by their error system
// ---------------------------------------------------------------------------

type ChainFamily int

const (
	ChainFamilyEVM ChainFamily = iota
	ChainFamilySolana
	ChainFamilyBitcoin
	ChainFamilyCosmosSDK
	ChainFamilyStarknet
	ChainFamilyAptos
	ChainFamilyNEAR
	ChainFamilyXRP
	ChainFamilyStellar
	ChainFamilyTON
	ChainFamilyPolygonZkEVM
	ChainFamilyCardano
	ChainFamilyTron
)

// chainFamilyMap maps chain ID strings (from spec JSON files) to their chain family.
// Sourced from specs/mainnet-1/specs/ and specs/testnet-2/specs/.
var chainFamilyMap = map[string]ChainFamily{
	// EVM — all jsonrpc chains with EVM execution semantics
	"ETH1": ChainFamilyEVM, "SEP1": ChainFamilyEVM, "HOL1": ChainFamilyEVM,
	"ARBITRUM": ChainFamilyEVM, "ARBITRUMN": ChainFamilyEVM, "ARBITRUMS": ChainFamilyEVM,
	"POLYGON": ChainFamilyEVM, "POLYGONA": ChainFamilyEVM,
	"BASE": ChainFamilyEVM, "BASES": ChainFamilyEVM,
	"OPTM": ChainFamilyEVM, "OPTMS": ChainFamilyEVM,
	"AVAX": ChainFamilyEVM, "AVAXT": ChainFamilyEVM,
	"AVAXC": ChainFamilyEVM, "AVAXCT": ChainFamilyEVM,
	"AVAXP": ChainFamilyEVM, "AVAXPT": ChainFamilyEVM,
	"BSC": ChainFamilyEVM, "BSCT": ChainFamilyEVM,
	"BLAST": ChainFamilyEVM, "BLASTSP": ChainFamilyEVM,
	"FTM250": ChainFamilyEVM, "FTM4002": ChainFamilyEVM,
	"SONIC": ChainFamilyEVM, "SONICT": ChainFamilyEVM,
	"CELO": ChainFamilyEVM, "ALFAJORES": ChainFamilyEVM,
	"FUSE": ChainFamilyEVM,
	"FVM":  ChainFamilyEVM, "FVMT": ChainFamilyEVM,
	"HEDERA": ChainFamilyEVM, "HEDERAT": ChainFamilyEVM,
	"HYPERLIQUID": ChainFamilyEVM, "HYPERLIQUIDT": ChainFamilyEVM,
	"WORLDCHAIN": ChainFamilyEVM, "WORLDCHAINS": ChainFamilyEVM,
	"AGR": ChainFamilyEVM, "AGRT": ChainFamilyEVM,
	"BERA": ChainFamilyEVM, "BERAT": ChainFamilyEVM, "BERAT2": ChainFamilyEVM,
	"CANTO":     ChainFamilyEVM,
	"KAKAROTT":  ChainFamilyEVM,
	"SPARK":     ChainFamilyEVM,
	"ETHBEACON": ChainFamilyEVM,
	"MOVEMENT":  ChainFamilyEVM, "MOVEMENTT": ChainFamilyEVM,

	// Solana
	"SOLANA": ChainFamilySolana, "SOLANAT": ChainFamilySolana,

	// Bitcoin/UTXO
	"BTC": ChainFamilyBitcoin, "BTCT": ChainFamilyBitcoin,
	"LTC": ChainFamilyBitcoin, "LTCT": ChainFamilyBitcoin,
	"DOGE": ChainFamilyBitcoin, "DOGET": ChainFamilyBitcoin,
	"BCH": ChainFamilyBitcoin, "BCHT": ChainFamilyBitcoin,

	// Cosmos SDK — chains with grpc/rest/tendermintrpc interfaces
	"COSMOSHUB": ChainFamilyCosmosSDK, "COSMOSHUBT": ChainFamilyCosmosSDK,
	"LAVA": ChainFamilyCosmosSDK, "LAV1": ChainFamilyCosmosSDK,
	"AXELAR": ChainFamilyCosmosSDK, "AXELART": ChainFamilyCosmosSDK,
	"EVMOS": ChainFamilyCosmosSDK, "EVMOST": ChainFamilyCosmosSDK,
	"OSMOSIS": ChainFamilyCosmosSDK, "OSMOSIST": ChainFamilyCosmosSDK,
	"JUN1": ChainFamilyCosmosSDK, "JUNT1": ChainFamilyCosmosSDK,
	"CELESTIA": ChainFamilyCosmosSDK, "CELESTIATA": ChainFamilyCosmosSDK, "CELESTIATM": ChainFamilyCosmosSDK,
	"STRGZ": ChainFamilyCosmosSDK, "STRGZT": ChainFamilyCosmosSDK,
	"SECRET": ChainFamilyCosmosSDK, "SECRETP": ChainFamilyCosmosSDK,
	"IBC":       ChainFamilyCosmosSDK,
	"COSMOSSDK": ChainFamilyCosmosSDK, "COSMOSSDK50": ChainFamilyCosmosSDK,
	"COSMOSSDK45DEP": ChainFamilyCosmosSDK, "COSMOSSDKFULL": ChainFamilyCosmosSDK,
	"COSMOSWASM": ChainFamilyCosmosSDK,
	"ETHERMINT":  ChainFamilyCosmosSDK,
	"TENDERMINT": ChainFamilyCosmosSDK,
	"UNIONT":     ChainFamilyCosmosSDK,

	// Starknet
	"STRK": ChainFamilyStarknet, "STRKS": ChainFamilyStarknet,

	// Aptos
	"APT1": ChainFamilyAptos, "SUIT": ChainFamilyAptos,

	// NEAR
	"NEAR": ChainFamilyNEAR, "NEART": ChainFamilyNEAR,

	// XRP
	"XRP": ChainFamilyXRP, "XRPT": ChainFamilyXRP,

	// Stellar
	"XLM": ChainFamilyStellar, "XLMT": ChainFamilyStellar,

	// TON
	"TON": ChainFamilyTON, "TONT": ChainFamilyTON,

	// Tron — REST-based with own error format (string enums like CONTRACT_EXE_ERROR)
	"TRX": ChainFamilyTron, "TRXT": ChainFamilyTron,

	// Cardano
	"CARDANO": ChainFamilyCardano, "CARDANOT": ChainFamilyCardano,

	// Koii
	"KOII": ChainFamilySolana, "KOIIT": ChainFamilySolana,
}

// GetChainFamily returns the chain family for a given chain ID.
// Returns ChainFamilyEVM as default for unknown chains (most common case).
func GetChainFamily(chainID string) (ChainFamily, bool) {
	family, ok := chainFamilyMap[chainID]
	return family, ok
}

// ---------------------------------------------------------------------------
// Transport types — determines which generic matchers to evaluate
// ---------------------------------------------------------------------------

type TransportType int

const (
	TransportJsonRPC TransportType = iota // EVM, Solana, Bitcoin, etc. (includes WebSocket subscriptions)
	TransportREST                         // Aptos, Stellar, some Cosmos endpoints
	TransportGRPC                         // Cosmos SDK chains
)

// ---------------------------------------------------------------------------
// Error Matchers
// ---------------------------------------------------------------------------

// ErrorMatcher classifies errors by matching error codes and/or messages.
type ErrorMatcher interface {
	Matches(errorCode int, errorMessage string) bool
}

// CodeEquals matches an exact error code (e.g., JSON-RPC -32601).
type codeEqualsMatcher struct {
	code int
}

func (m codeEqualsMatcher) Matches(errorCode int, _ string) bool {
	return errorCode == m.code
}

func CodeEquals(code int) ErrorMatcher {
	return codeEqualsMatcher{code: code}
}

// MessageContains matches a case-insensitive substring in the error message.
type messageContainsMatcher struct {
	substring string // pre-lowercased at construction time
}

func (m messageContainsMatcher) Matches(_ int, errorMessage string) bool {
	return strings.Contains(strings.ToLower(errorMessage), m.substring)
}

func MessageContains(substring string) ErrorMatcher {
	return messageContainsMatcher{substring: strings.ToLower(substring)}
}

// MessageRegex matches the error message against a regex pattern.
type messageRegexMatcher struct {
	re *regexp.Regexp
}

func (m messageRegexMatcher) Matches(_ int, errorMessage string) bool {
	return m.re.MatchString(errorMessage)
}

func MessageRegex(pattern string) ErrorMatcher {
	return messageRegexMatcher{re: regexp.MustCompile(pattern)}
}

// HTTPStatusContains matches an HTTP status code in the error message with non-digit
// boundary checks. This prevents false positives like "500" matching inside "25001500".
// Prefer structured status codes (CodeEquals) when the HTTP status is available as an int.
type httpStatusMatcher struct {
	status string
}

func (m httpStatusMatcher) Matches(_ int, errorMessage string) bool {
	target := m.status
	msg := errorMessage
	for {
		idx := strings.Index(msg, target)
		if idx < 0 {
			return false
		}
		// Check that the character before the match (if any) is not a digit
		if idx > 0 && msg[idx-1] >= '0' && msg[idx-1] <= '9' {
			msg = msg[idx+len(target):]
			continue
		}
		// Check that the character after the match (if any) is not a digit
		end := idx + len(target)
		if end < len(msg) && msg[end] >= '0' && msg[end] <= '9' {
			msg = msg[end:]
			continue
		}
		return true
	}
}

func HTTPStatusContains(status int) ErrorMatcher {
	return httpStatusMatcher{status: fmt.Sprintf("%d", status)}
}

// GRPCCodeEquals matches a gRPC status code.
type grpcCodeMatcher struct {
	code uint32
}

func (m grpcCodeMatcher) Matches(errorCode int, _ string) bool {
	return uint32(errorCode) == m.code
}

func GRPCCodeEquals(code uint32) ErrorMatcher {
	return grpcCodeMatcher{code: code}
}

// ---------------------------------------------------------------------------
// Error mapping entry
// ---------------------------------------------------------------------------

type errorMapping struct {
	Matcher   ErrorMatcher
	LavaError *LavaError
}

// ---------------------------------------------------------------------------
// Reserved error codes
// ---------------------------------------------------------------------------

// LavaErrorUnknown is the fallback for unclassified errors.
// Category is External because unmatched errors are node pass-throughs.
var LavaErrorUnknown = &LavaError{
	Code: 0, Name: "UNKNOWN_ERROR", Category: CategoryExternal,
	Description: "Unclassified error — no matcher matched", Retryable: true,
}

// ---------------------------------------------------------------------------
// Registry — unexported, access via lookup helpers
// ---------------------------------------------------------------------------

var errorRegistry = map[uint32]*LavaError{
	LavaErrorUnknown.Code: LavaErrorUnknown,
}

func registerError(le *LavaError) *LavaError {
	if _, exists := errorRegistry[le.Code]; exists {
		panic(fmt.Sprintf("duplicate error code: %d (%s)", le.Code, le.Name))
	}
	errorRegistry[le.Code] = le
	return le
}

// ---------------------------------------------------------------------------
// Lookup helpers
// ---------------------------------------------------------------------------

func getLavaError(code uint32) *LavaError {
	if le, ok := errorRegistry[code]; ok {
		return le
	}
	return LavaErrorUnknown
}

func getLavaErrorByName(name string) *LavaError {
	for _, le := range errorRegistry {
		if le.Name == name {
			return le
		}
	}
	return LavaErrorUnknown
}

func isRetryable(code uint32) bool {
	return getLavaError(code).Retryable
}

// LavaWrappedError wraps a LavaError with context, supporting errors.Is matching.
type LavaWrappedError struct {
	LavaErr *LavaError
	Context string
}

func (e *LavaWrappedError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s: %s", e.Context, e.LavaErr.Description)
	}
	return e.LavaErr.Error()
}

func (e *LavaWrappedError) Is(target error) bool {
	if t, ok := target.(*LavaError); ok {
		return e.LavaErr.Code == t.Code
	}
	return false
}

func (e *LavaWrappedError) Unwrap() error {
	return e.LavaErr
}

// NewLavaError creates an error that wraps a LavaError with optional context.
// Supports errors.Is(err, LavaErrorSomething) matching by code.
func NewLavaError(lavaErr *LavaError, context string) error {
	return &LavaWrappedError{LavaErr: lavaErr, Context: context}
}

func IsInternal(code uint32) bool {
	return getLavaError(code).Category == CategoryInternal
}

func IsExternal(code uint32) bool {
	return getLavaError(code).Category == CategoryExternal
}
