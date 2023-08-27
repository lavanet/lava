package v2

func (p *PolicyV2) Equal(that interface{}) bool {
	if that == nil {
		return p == nil
	}

	that1, ok := that.(*PolicyV2)
	if !ok {
		that2, ok := that.(PolicyV2)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return p == nil
	} else if p == nil {
		return false
	}
	if len(p.ChainPolicies) != len(that1.ChainPolicies) {
		return false
	}
	for i := range p.ChainPolicies {
		if !p.ChainPolicies[i].Equal(&that1.ChainPolicies[i]) {
			return false
		}
	}
	if p.GeolocationProfile != that1.GeolocationProfile {
		return false
	}
	if p.TotalCuLimit != that1.TotalCuLimit {
		return false
	}
	if p.EpochCuLimit != that1.EpochCuLimit {
		return false
	}
	if p.MaxProvidersToPair != that1.MaxProvidersToPair {
		return false
	}

	return true
}

func (cp *ChainPolicyV2) Equal(that interface{}) bool {
	if that == nil {
		return cp == nil
	}

	that1, ok := that.(*ChainPolicyV2)
	if !ok {
		that2, ok := that.(ChainPolicyV2)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return cp == nil
	} else if cp == nil {
		return false
	}
	if cp.ChainId != that1.ChainId {
		return false
	}
	if len(cp.Apis) != len(that1.Apis) {
		return false
	}
	for i := range cp.Apis {
		if cp.Apis[i] != that1.Apis[i] {
			return false
		}
	}

	return true
}
