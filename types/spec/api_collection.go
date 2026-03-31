package spec

import (
	"fmt"
)

// ---------------------------------------------------------------------------
// Struct definitions
// ---------------------------------------------------------------------------

// ApiCollection groups a set of APIs and associated metadata under a single
// CollectionData key (interface + path + type + add-on).
type ApiCollection struct {
	Enabled         bool              `json:"enabled,omitempty"`
	CollectionData  CollectionData    `json:"collection_data"`
	Apis            []*Api            `json:"apis,omitempty"`
	Headers         []*Header         `json:"headers,omitempty"`
	InheritanceApis []*CollectionData `json:"inheritance_apis,omitempty"`
	ParseDirectives []*ParseDirective `json:"parse_directives,omitempty"`
	Extensions      []*Extension      `json:"extensions,omitempty"`
	Verifications   []*Verification   `json:"verifications,omitempty"`
}

// CollectionData is the composite key that uniquely identifies an
// ApiCollection within a Spec.
type CollectionData struct {
	ApiInterface string `json:"api_interface" mapstructure:"api_interface"`
	InternalPath string `json:"internal_path" mapstructure:"internal_path"`
	Type         string `json:"type"          mapstructure:"type"`
	AddOn        string `json:"add_on"        mapstructure:"add_on"`
}

// String returns a human-readable representation used for sorting and
// error messages.
func (cd CollectionData) String() string {
	return fmt.Sprintf("{apiInterface:%s internalPath:%s type:%s addOn:%s}",
		cd.ApiInterface, cd.InternalPath, cd.Type, cd.AddOn)
}

// Api describes a single RPC method or REST endpoint exposed by a provider.
type Api struct {
	Enabled           bool          `json:"enabled,omitempty"            mapstructure:"enabled"`
	Name              string        `json:"name"                         mapstructure:"name"`
	ComputeUnits      uint64        `json:"compute_units,omitempty"`
	ExtraComputeUnits uint64        `json:"extra_compute_units,omitempty"`
	Category          SpecCategory  `json:"category"`
	BlockParsing      BlockParser   `json:"block_parsing"`
	TimeoutMs         uint64        `json:"timeout_ms,omitempty"`
	Parsers           []GenericParser `json:"parsers,omitempty"`
}

// Header describes an HTTP header that should be forwarded, rewritten, or
// stripped during relay processing.
type Header struct {
	Name        string            `json:"name"`
	Kind        Header_HeaderType `json:"kind,omitempty"`
	FunctionTag FUNCTION_TAG      `json:"function_tag,omitempty"`
	Value       string            `json:"value,omitempty"`
}

// Extension attaches additional rules (e.g. archive block depth) to an
// ApiCollection and adjusts the compute-unit cost via CuMultiplier.
type Extension struct {
	Name         string `json:"name"`
	Rule         *Rule  `json:"rule,omitempty"`
	CuMultiplier uint64 `json:"cu_multiplier,omitempty"`
}

// Rule defines a threshold block number used by Extension.
type Rule struct {
	Block uint64 `json:"block,omitempty"`
}

// ParseDirective maps a FUNCTION_TAG to a request template and result parser,
// enabling the node client to construct and interpret specific RPC calls.
type ParseDirective struct {
	FunctionTag      FUNCTION_TAG    `json:"function_tag,omitempty"`
	FunctionTemplate string          `json:"function_template,omitempty"`
	ResultParsing    BlockParser     `json:"result_parsing"`
	ApiName          string          `json:"api_name,omitempty"`
	Parsers          []GenericParser `json:"parsers,omitempty"`
}

// GenericParser extracts a value (e.g. a block number) from an RPC response
// using a JSONPath-like parse_path and an optional rule.
type GenericParser struct {
	ParsePath string      `json:"parse_path,omitempty"`
	Value     string      `json:"value,omitempty"`
	ParseType PARSER_TYPE `json:"parse_type,omitempty"`
	Rule      string      `json:"rule,omitempty"`
}

// BlockParser configures how a block number is extracted from a request or
// response, including optional encoding (base64 / hex).
type BlockParser struct {
	ParserArg    []string    `json:"parser_arg,omitempty"`
	ParserFunc   PARSER_FUNC `json:"parser_func,omitempty"`
	DefaultValue string      `json:"default_value,omitempty"`
	Encoding     string      `json:"encoding,omitempty"`
}

// SpecCategory classifies API behaviour (determinism, statefulness, etc.) and
// is used to select appropriate providers and routing strategies.
type SpecCategory struct {
	Deterministic bool   `json:"deterministic,omitempty"`
	Local         bool   `json:"local,omitempty"`
	Subscription  bool   `json:"subscription,omitempty"`
	Stateful      uint32 `json:"stateful,omitempty"`
	HangingApi    bool   `json:"hanging_api,omitempty"`
}

// Verification pairs a ParseDirective with a set of expected values, enabling
// the consumer to verify provider responses against known-good answers.
type Verification struct {
	Name           string          `json:"name"`
	ParseDirective *ParseDirective `json:"parse_directive,omitempty"`
	Values         []*ParseValue   `json:"values,omitempty"`
}

// ParseValue holds a single expected response value for a Verification,
// optionally scoped to a named extension and with configurable severity.
type ParseValue struct {
	Extension      string                          `json:"extension,omitempty"`
	ExpectedValue  string                          `json:"expected_value,omitempty"`
	LatestDistance uint64                          `json:"latest_distance,omitempty"`
	Severity       ParseValue_VerificationSeverity `json:"severity,omitempty"`
}

// ---------------------------------------------------------------------------
// ApiCollection getters
// ---------------------------------------------------------------------------

func (m *ApiCollection) GetEnabled() bool {
	if m != nil {
		return m.Enabled
	}
	return false
}

func (m *ApiCollection) GetCollectionData() CollectionData {
	if m != nil {
		return m.CollectionData
	}
	return CollectionData{}
}

func (m *ApiCollection) GetApis() []*Api {
	if m != nil {
		return m.Apis
	}
	return nil
}

func (m *ApiCollection) GetHeaders() []*Header {
	if m != nil {
		return m.Headers
	}
	return nil
}

func (m *ApiCollection) GetInheritanceApis() []*CollectionData {
	if m != nil {
		return m.InheritanceApis
	}
	return nil
}

func (m *ApiCollection) GetParseDirectives() []*ParseDirective {
	if m != nil {
		return m.ParseDirectives
	}
	return nil
}

func (m *ApiCollection) GetExtensions() []*Extension {
	if m != nil {
		return m.Extensions
	}
	return nil
}

func (m *ApiCollection) GetVerifications() []*Verification {
	if m != nil {
		return m.Verifications
	}
	return nil
}

// ---------------------------------------------------------------------------
// CollectionData getters
// ---------------------------------------------------------------------------

func (m *CollectionData) GetApiInterface() string {
	if m != nil {
		return m.ApiInterface
	}
	return ""
}

func (m *CollectionData) GetInternalPath() string {
	if m != nil {
		return m.InternalPath
	}
	return ""
}

func (m *CollectionData) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *CollectionData) GetAddOn() string {
	if m != nil {
		return m.AddOn
	}
	return ""
}

// ---------------------------------------------------------------------------
// Api getters
// ---------------------------------------------------------------------------

func (m *Api) GetEnabled() bool {
	if m != nil {
		return m.Enabled
	}
	return false
}

func (m *Api) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Api) GetComputeUnits() uint64 {
	if m != nil {
		return m.ComputeUnits
	}
	return 0
}

func (m *Api) GetExtraComputeUnits() uint64 {
	if m != nil {
		return m.ExtraComputeUnits
	}
	return 0
}

func (m *Api) GetCategory() SpecCategory {
	if m != nil {
		return m.Category
	}
	return SpecCategory{}
}

func (m *Api) GetBlockParsing() BlockParser {
	if m != nil {
		return m.BlockParsing
	}
	return BlockParser{}
}

func (m *Api) GetTimeoutMs() uint64 {
	if m != nil {
		return m.TimeoutMs
	}
	return 0
}

func (m *Api) GetParsers() []GenericParser {
	if m != nil {
		return m.Parsers
	}
	return nil
}

// ---------------------------------------------------------------------------
// Header getters
// ---------------------------------------------------------------------------

func (m *Header) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Header) GetKind() Header_HeaderType {
	if m != nil {
		return m.Kind
	}
	return Header_pass_send
}

func (m *Header) GetFunctionTag() FUNCTION_TAG {
	if m != nil {
		return m.FunctionTag
	}
	return FUNCTION_TAG_DISABLED
}

func (m *Header) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

// ---------------------------------------------------------------------------
// Extension getters
// ---------------------------------------------------------------------------

func (m *Extension) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Extension) GetRule() *Rule {
	if m != nil {
		return m.Rule
	}
	return nil
}

func (m *Extension) GetCuMultiplier() uint64 {
	if m != nil {
		return m.CuMultiplier
	}
	return 0
}

// ---------------------------------------------------------------------------
// Rule getters
// ---------------------------------------------------------------------------

func (m *Rule) GetBlock() uint64 {
	if m != nil {
		return m.Block
	}
	return 0
}

// ---------------------------------------------------------------------------
// ParseDirective getters
// ---------------------------------------------------------------------------

func (m *ParseDirective) GetFunctionTag() FUNCTION_TAG {
	if m != nil {
		return m.FunctionTag
	}
	return FUNCTION_TAG_DISABLED
}

func (m *ParseDirective) GetFunctionTemplate() string {
	if m != nil {
		return m.FunctionTemplate
	}
	return ""
}

func (m *ParseDirective) GetResultParsing() BlockParser {
	if m != nil {
		return m.ResultParsing
	}
	return BlockParser{}
}

func (m *ParseDirective) GetApiName() string {
	if m != nil {
		return m.ApiName
	}
	return ""
}

func (m *ParseDirective) GetParsers() []GenericParser {
	if m != nil {
		return m.Parsers
	}
	return nil
}

// ---------------------------------------------------------------------------
// GenericParser getters
// ---------------------------------------------------------------------------

func (m *GenericParser) GetParsePath() string {
	if m != nil {
		return m.ParsePath
	}
	return ""
}

func (m *GenericParser) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

func (m *GenericParser) GetParseType() PARSER_TYPE {
	if m != nil {
		return m.ParseType
	}
	return PARSER_TYPE_NO_PARSER
}

func (m *GenericParser) GetRule() string {
	if m != nil {
		return m.Rule
	}
	return ""
}

// ---------------------------------------------------------------------------
// BlockParser getters
// ---------------------------------------------------------------------------

func (m *BlockParser) GetParserArg() []string {
	if m != nil {
		return m.ParserArg
	}
	return nil
}

func (m *BlockParser) GetParserFunc() PARSER_FUNC {
	if m != nil {
		return m.ParserFunc
	}
	return PARSER_FUNC_EMPTY
}

func (m *BlockParser) GetDefaultValue() string {
	if m != nil {
		return m.DefaultValue
	}
	return ""
}

func (m *BlockParser) GetEncoding() string {
	if m != nil {
		return m.Encoding
	}
	return ""
}

// ---------------------------------------------------------------------------
// SpecCategory getters
// ---------------------------------------------------------------------------

func (m *SpecCategory) GetDeterministic() bool {
	if m != nil {
		return m.Deterministic
	}
	return false
}

func (m *SpecCategory) GetLocal() bool {
	if m != nil {
		return m.Local
	}
	return false
}

func (m *SpecCategory) GetSubscription() bool {
	if m != nil {
		return m.Subscription
	}
	return false
}

func (m *SpecCategory) GetStateful() uint32 {
	if m != nil {
		return m.Stateful
	}
	return 0
}

func (m *SpecCategory) GetHangingApi() bool {
	if m != nil {
		return m.HangingApi
	}
	return false
}

// Combine merges two SpecCategory values following the spec inheritance rules:
// Deterministic is AND-ed (both must agree), all boolean flags are OR-ed, and
// Stateful takes the maximum value.
func (sc SpecCategory) Combine(other SpecCategory) SpecCategory {
	return SpecCategory{
		Deterministic: sc.Deterministic && other.Deterministic,
		Local:         sc.Local || other.Local,
		Subscription:  sc.Subscription || other.Subscription,
		Stateful:      max(sc.Stateful, other.Stateful),
		HangingApi:    sc.HangingApi || other.HangingApi,
	}
}

// ---------------------------------------------------------------------------
// Verification getters
// ---------------------------------------------------------------------------

func (m *Verification) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Verification) GetParseDirective() *ParseDirective {
	if m != nil {
		return m.ParseDirective
	}
	return nil
}

func (m *Verification) GetValues() []*ParseValue {
	if m != nil {
		return m.Values
	}
	return nil
}

// ---------------------------------------------------------------------------
// ParseValue getters
// ---------------------------------------------------------------------------

func (m *ParseValue) GetExtension() string {
	if m != nil {
		return m.Extension
	}
	return ""
}

func (m *ParseValue) GetExpectedValue() string {
	if m != nil {
		return m.ExpectedValue
	}
	return ""
}

func (m *ParseValue) GetLatestDistance() uint64 {
	if m != nil {
		return m.LatestDistance
	}
	return 0
}

func (m *ParseValue) GetSeverity() ParseValue_VerificationSeverity {
	if m != nil {
		return m.Severity
	}
	return ParseValue_Fail
}

// ---------------------------------------------------------------------------
// ApiCollection expansion / inheritance logic
// (ported from original x/spec/types/api_collection.go without Cosmos deps)
// ---------------------------------------------------------------------------

// CanExpand reports whether cd can inherit from other, i.e. both share the
// same ApiInterface and Type, or other acts as a base collection with an empty
// ApiInterface.
func (cd *CollectionData) CanExpand(other *CollectionData) bool {
	return cd.ApiInterface == other.ApiInterface && cd.Type == other.Type || other.ApiInterface == ""
}

// Expand resolves InheritanceApis for this collection by recursively
// expanding its dependencies from myCollections, then merging their content.
func (apic *ApiCollection) Expand(myCollections map[CollectionData]*ApiCollection, dependencies map[CollectionData]struct{}) error {
	dependencies[apic.CollectionData] = struct{}{}
	defer delete(dependencies, apic.CollectionData)

	inheritanceApis := apic.InheritanceApis
	apic.InheritanceApis = []*CollectionData{} // clear to avoid double-expansion

	var relevantCollections []*ApiCollection
	for _, inheritingCollection := range inheritanceApis {
		collection, ok := myCollections[*inheritingCollection]
		if !ok {
			return fmt.Errorf("did not find inheritingCollection in myCollections %v", inheritingCollection)
		}
		if !apic.CollectionData.CanExpand(&collection.CollectionData) {
			return fmt.Errorf("invalid inheriting collection %v", inheritingCollection)
		}
		if _, ok := dependencies[collection.CollectionData]; ok {
			return fmt.Errorf("circular dependency in inheritance, %v", collection)
		}
		if err := collection.Expand(myCollections, dependencies); err != nil {
			return err
		}
		relevantCollections = append(relevantCollections, collection)
	}
	// expand is within the same spec → allow combining with disabled collections
	return apic.CombineWithOthers(relevantCollections, true, true)
}

// Inherit merges the already-expanded parent collections into this collection
// without tracking circular dependencies (cross-spec inheritance is allowed to
// share the same collection type).
func (apic *ApiCollection) Inherit(relevantCollections []*ApiCollection, _ map[CollectionData]struct{}) error {
	return apic.CombineWithOthers(relevantCollections, false, true)
}

// Equals reports whether two ApiCollections share the same CollectionData key.
func (apic *ApiCollection) Equals(other *ApiCollection) bool {
	return other.CollectionData == apic.CollectionData
}

// InheritAllFields first expands internal inheritance for this collection and
// then merges all parent collections into it.
func (apic *ApiCollection) InheritAllFields(myCollections map[CollectionData]*ApiCollection, relevantParentCollections []*ApiCollection) error {
	for _, other := range relevantParentCollections {
		if !apic.Equals(other) {
			return fmt.Errorf("incompatible inheritance, apiCollections aren't equal %v", apic)
		}
	}
	if err := apic.Expand(myCollections, map[CollectionData]struct{}{}); err != nil {
		return err
	}
	return apic.Inherit(relevantParentCollections, map[CollectionData]struct{}{})
}

// CombineWithOthers merges APIs, headers, parse directives, extensions, and
// verifications from others into apic. combineWithDisabled controls whether
// disabled collections are still merged; allowOverwrite controls whether
// duplicate keys cause errors.
func (apic *ApiCollection) CombineWithOthers(others []*ApiCollection, combineWithDisabled, allowOverwrite bool) (err error) {
	if apic == nil {
		return fmt.Errorf("CombineWithOthers: API collection is nil")
	}

	mergedApis := map[string]interface{}{}
	mergedHeaders := map[string]interface{}{}
	mergedParsers := map[string]interface{}{}
	mergedExtensions := map[string]interface{}{}
	mergedVerifications := map[string]interface{}{}

	mergedApisList := []*Api{}
	mergedHeadersList := []*Header{}
	mergedParsersList := []*ParseDirective{}
	mergedExtensionsList := []*Extension{}
	mergedVerificationsList := []*Verification{}

	currentApis := GetCurrentFromCombinable(apic.Apis)
	currentHeaders := GetCurrentFromCombinable(apic.Headers)
	currentParsers := GetCurrentFromCombinable(apic.ParseDirectives)
	currentExtensions := GetCurrentFromCombinable(apic.Extensions)
	currentVerifications := GetCurrentFromCombinable(apic.Verifications)

	for _, collection := range others {
		if !collection.Enabled && !combineWithDisabled {
			continue
		}

		mergedApisList, mergedApis, err = CombineFields(currentApis, collection.Apis, mergedApis, mergedApisList, allowOverwrite)
		if err != nil {
			return fmt.Errorf("merging apis error %w, %v other collection %v", err, apic, collection.CollectionData)
		}

		mergedHeadersList, mergedHeaders, err = CombineFields(currentHeaders, collection.Headers, mergedHeaders, mergedHeadersList, allowOverwrite)
		if err != nil {
			return fmt.Errorf("merging headers error %w, %v other collection %v", err, apic, collection.CollectionData)
		}

		mergedParsersList, mergedParsers, err = CombineFields(currentParsers, collection.ParseDirectives, mergedParsers, mergedParsersList, allowOverwrite)
		if err != nil {
			return fmt.Errorf("merging parse directives error %w, %v other collection %v", err, apic, collection.CollectionData)
		}

		mergedExtensionsList, mergedExtensions, err = CombineFields(currentExtensions, collection.Extensions, mergedExtensions, mergedExtensionsList, allowOverwrite)
		if err != nil {
			return fmt.Errorf("merging extensions error %w, %v other collection %v", err, apic, collection.CollectionData)
		}

		mergedVerificationsList, mergedVerifications, err = CombineFields(currentVerifications, collection.Verifications, mergedVerifications, mergedVerificationsList, allowOverwrite)
		if err != nil {
			return fmt.Errorf("merging verifications error %w, %v other collection %v", err, apic, collection.CollectionData)
		}
	}

	apic.Apis, err = CombineUnique(mergedApisList, apic.Apis, currentApis, allowOverwrite)
	if err != nil {
		return fmt.Errorf("error %w in apis combination in collection %#v", err, apic)
	}

	apic.Headers, err = CombineUnique(mergedHeadersList, apic.Headers, currentHeaders, allowOverwrite)
	if err != nil {
		return fmt.Errorf("error %w in headers combination in collection %#v", err, apic)
	}

	apic.ParseDirectives, err = CombineUnique(mergedParsersList, apic.ParseDirectives, currentParsers, allowOverwrite)
	if err != nil {
		return fmt.Errorf("error %w in parse directive combination in collection %#v", err, apic)
	}

	apic.Extensions, err = CombineUnique(mergedExtensionsList, apic.Extensions, currentExtensions, allowOverwrite)
	if err != nil {
		return fmt.Errorf("error %w in extension combination in collection %#v", err, apic)
	}

	apic.Verifications, err = CombineUnique(mergedVerificationsList, apic.Verifications, currentVerifications, allowOverwrite)
	if err != nil {
		return fmt.Errorf("error %w in verification combination in collection %#v", err, apic)
	}

	return nil
}
