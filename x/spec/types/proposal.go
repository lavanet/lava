package types

// SpecAddProposal represents a governance proposal to add or update one or
// more specs.  It mirrors the original protobuf-generated type that was
// removed together with the blockchain modules.
type SpecAddProposal struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Specs       []Spec `json:"specs"`
}

// SpecAddProposalJSON is the on-disk JSON envelope used by spec proposal files.
// It is the authoritative format understood by the spec fetcher and loader.
type SpecAddProposalJSON struct {
	Proposal SpecAddProposal `json:"proposal"`
	Deposit  string          `json:"deposit,omitempty" yaml:"deposit,omitempty"`
}
