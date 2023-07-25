package types

func (rm ReplyMetadata) GetSignature() []byte {
	return rm.Sig
}

func (rm ReplyMetadata) DataToSign() []byte {
	return rm.HashAllDataHash
}

func (rm ReplyMetadata) HashCount() int {
	return 0
}
