package types

import "errors"

func (specN *SpecName) ValidateBasic() error {
	if len(specN.Name) > 100 {
		return errors.New("invalid spec name string, length too big")
	}
	return nil
}
