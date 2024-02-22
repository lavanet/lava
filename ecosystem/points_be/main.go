package main

import (
	"cosmos-sdk/crypto/keys/secp256k1"
	"cosmos-sdk/types/bech32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var pubkeyType, pubkeyValue, signature, nonce string

	var rootCmd = &cobra.Command{
		Use:   "./points_be --nonce \"SomeRandomNonce\" --pubkey_type \"tendermint/PubKeySecp256k1\" --pubkey_value \"A4lWTKNYKEpXkkqsTtvy/2JogKr3BPcIvKRmuhaaMxcY\" --signature \"CPNPl/0dQR8IeVzMJ8ynB152FrpUa1878pTsUoKiYv5XDQhVnkW32O1jwcnnME1CLKnSR/fuBx5dhgCj/ngaUw==\"",
		Short: "CLI tool for signature verification",
		Long: `This CLI tool allows you to verify signatures using various public key types.
The signature is verified against the provided public key and nonce.`,
		Run: func(cmd *cobra.Command, args []string) {
			if pubkeyType == "" || pubkeyValue == "" || signature == "" || nonce == "" {
				fmt.Println("Error: pubkey_type, pubkey_value, signature, and nonce flags are required for verification.")
				os.Exit(1)
			}
			err := verifySignatureAndCompare(pubkeyType, pubkeyValue, signature, nonce)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&pubkeyType, "pubkey_type", "", "Type of the public key (e.g., secp256k1)")
	rootCmd.Flags().StringVar(&pubkeyValue, "pubkey_value", "", "Public key value of the signer")
	rootCmd.Flags().StringVar(&signature, "signature", "", "Signature to be verified")
	rootCmd.Flags().StringVar(&nonce, "nonce", "", "Nonce that was signed")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func parsePubKeyJSON(pubKeyType, pubKeyValue string) (*secp256k1.PubKey, error) {
	// Decode the base64-encoded value to get the raw bytes
	decodedValue, err := base64.StdEncoding.DecodeString(pubKeyValue)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	pk := struct {
		Type  string `json:"type"`
		Value []byte `json:"value"`
	}{Type: pubKeyType, Value: decodedValue}
	if pk.Type != "tendermint/PubKeySecp256k1" {
		return nil, errors.New("invalid pubkey type")
	}

	pubkey := secp256k1.PubKey{}
	if err := pubkey.UnmarshalAmino(pk.Value); err != nil {
		return nil, errors.New("failed to unmarshal amino pubkey")
	}
	return &pubkey, nil
}

func makeADR36SignDoc(data []byte, signerAddress string) []byte {
	const template = `{"account_number":"0","chain_id":"","fee":{"amount":[],"gas":"0"},"memo":"","msgs":[{"type":"sign/MsgSignData","value":{"data":%s,"signer":%s}}],"sequence":"0"}`
	dataJSON, err := json.Marshal(base64.StdEncoding.EncodeToString(data))
	if err != nil {
		panic("failed to marshal json string, something is wrong with the JSON encoder")
	}
	signerJSON, err := json.Marshal(signerAddress)
	if err != nil {
		panic("failed to marshal json string, something is wrong with the JSON encoder")
	}
	return []byte(fmt.Sprintf(template, string(dataJSON), string(signerJSON)))
}

func verifySignatureAndCompare(pubkeyType, pubkeyValue, signedToken, nonce string) error {
	// Decode the base64 signature
	// Decode base64-encoded signature
	fmt.Println("Got Signature to validate")
	fmt.Println("Input Account", pubkeyType, pubkeyValue)
	fmt.Println("Signature", signedToken)

	signature, err := base64.StdEncoding.DecodeString(signedToken) // we use std encoding because keplr encode with std
	if err != nil {
		fmt.Println("failed to decode signature", err)
		return errors.New("failed to decode signature")
	}
	pubKey, err := parsePubKeyJSON(pubkeyType, pubkeyValue)
	if err != nil {
		fmt.Println("failed to decode parsePubKeyJSON", err)
		return errors.New("failed to decode parsePubKeyJSON")
	}
	addressBytes := pubKey.Address()
	fmt.Println("addressBytes", addressBytes)

	chainUserAddress, err := bech32.ConvertAndEncode("cosmos", addressBytes)
	if err != nil {
		fmt.Println("failed to decode signature", err)
		return errors.New("failed to encode bech32 address")
	}
	fmt.Println("nonceEncoded", []byte(nonce))
	fmt.Println("chainUserAddress", chainUserAddress)
	result := pubKey.VerifySignature(makeADR36SignDoc([]byte(nonce), chainUserAddress), signature)

	fmt.Println("result:", result)
	if result {
		fmt.Println("Signature valid")
		os.Exit(0)
	} else {
		fmt.Println("Signature invalid")
		os.Exit(1)
	}
	return nil
}
