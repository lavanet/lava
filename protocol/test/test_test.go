package test

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// used as a testing wrapper to just test fun stuff quickly.

func TestFlow(t *testing.T) {
	// Decoding base64 string
	bodyString := "CiEyNTAwMDAwMC4wMDAwMDAwMDAwMDAwMDAwMDBhZXZtb3M="
	decodedBody, err := base64.StdEncoding.DecodeString(bodyString)
	require.Nil(t, err)
	fmt.Println("Decoded body:", string(decodedBody)) // Output the decoded body
	resultString := strings.ReplaceAll(string(decodedBody), "!", "")
	fmt.Println("Result:", resultString) // Output the decoded body
}
