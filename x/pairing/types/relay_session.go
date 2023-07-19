package types

func (rs RelaySession) GetSignature() []byte {
	return rs.Sig
}

func (rs RelaySession) PrepareForSignature() []byte {
	rs.Badge = nil // its not a part of the signature, its a separate part
	rs.Sig = []byte{}
	return []byte(rs.String())
}
