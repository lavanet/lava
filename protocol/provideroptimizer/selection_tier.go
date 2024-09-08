package provideroptimizer

type SelectionTier interface {
}

type SelectionTierInst struct {
}

func NewSelectionTier(scores map[string]float64, tiers int) SelectionTier {
	return &SelectionTierInst{}
}
