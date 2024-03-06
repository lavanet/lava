package types

import fmt "fmt"

// validateVersion validates the Version param
func (v Version) validateVersion() error {
	newProviderTarget, err := versionToInteger(v.ProviderTarget)
	if err != nil {
		return fmt.Errorf("provider target version: %w", err)
	}
	newProviderMin, err := versionToInteger(v.ProviderMin)
	if err != nil {
		return fmt.Errorf("provider min version: %w", err)
	}
	newConsumerTarget, err := versionToInteger(v.ConsumerTarget)
	if err != nil {
		return fmt.Errorf("consumer target version: %w", err)
	}
	newConsumerMin, err := versionToInteger(v.ConsumerMin)
	if err != nil {
		return fmt.Errorf("consumer min version: %w", err)
	}

	// min version may not exceed target version
	if newProviderMin > newProviderTarget {
		return fmt.Errorf("provider min version exceeds target version: %d > %d",
			newProviderMin, newProviderTarget)
	}
	if newConsumerMin > newConsumerTarget {
		return fmt.Errorf("consumer min version exceeds target version: %d > %d",
			newConsumerMin, newConsumerTarget)
	}

	// provider and consumer versions must match (for now)
	if newProviderTarget != newConsumerTarget {
		return fmt.Errorf("provider and consumer target versions mismatch: %d != %d",
			newProviderTarget, newConsumerTarget)
	}
	if newProviderMin != newConsumerMin {
		return fmt.Errorf("provider and consumer min versions mismatch: %d != %d",
			newProviderMin, newConsumerMin)
	}

	return nil
}
