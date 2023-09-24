package parser

func CapStringLen(inp string) string {
	if len(inp) > 250 {
		return inp[:150] + "...Truncated..." + inp[len(inp)-100:]
	}
	return inp
}
