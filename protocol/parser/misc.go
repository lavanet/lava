package parser

func CapStringLen(inp string) string {
	if len(inp) > 200 {
		return inp[:100] + "...Truncated..." + inp[len(inp)-100:]
	}
	return inp
}
