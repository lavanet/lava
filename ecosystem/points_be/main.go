package main

import (
	"cosmos-sdk/crypto/keys/secp256k1"
	"cosmos-sdk/types/bech32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type RelaySession struct {
	Sig  []byte
	Data []byte
}

func (rs RelaySession) GetSignature() []byte {
	return rs.Sig
}

func (rs RelaySession) DataToSign() []byte {
	return rs.Data
}

func (rs RelaySession) HashRounds() int {
	return 1
}

var nonce = "SomeRandomNonce" // Global nonce value

func main() {
	app := fiber.New()

	// CORS middleware
	app.Use(func(c *fiber.Ctx) error {
		// Set the request origin as the allowed origin
		c.Set("Access-Control-Allow-Origin", c.Get("Origin"))

		// Handle preflight requests directly
		if c.Method() == "OPTIONS" {
			// Set up all allowed methods
			c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			// Allow headers
			c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			// Allow credentials
			c.Set("Access-Control-Allow-Credentials", "true")
			// Cache preflight request for 24 hours (in seconds)
			c.Set("Access-Control-Max-Age", "86400")
			return c.SendStatus(fiber.StatusNoContent)
		}

		if c.Method() == "DELETE" {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	})

	// Route definition
	app.Post("*", loginHandler)
	app.Post("/accounts/cosmos/login/", loginHandler)

	// Start the server
	port := 8080
	fmt.Println("Server listening on port %d...\n", port)
	if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
		fmt.Println("Error:", err)
	}
}

func loginHandler(c *fiber.Ctx) error {
	c.Set("Access-Control-Allow-Origin", c.Get("Origin"))
	// Set up all allowed methods
	c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	// Allow headers
	c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	// Allow credentials
	c.Set("Access-Control-Allow-Credentials", "true")
	// Cache preflight request for 24 hours (in seconds)
	c.Set("Access-Control-Max-Age", "86400")

	fmt.Println("GOT CALL")
	var data map[string]string
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	process := data["process"]
	switch process {
	case "token":
		fmt.Println("GOT CALL Token")
		// Return the nonce
		// var nonce = []byte("SomeRandomNonce") // Global nonce value
		response := fiber.Map{"data": nonce}
		return c.JSON(response)
	case "verify":
		fmt.Println("GOT CALL Verify")
		pubKeyType := data["pubkey_type"]
		pubKeyValue := data["pubkey_value"]
		signedToken := data["login_token"]
		// inviteCode := data["invite_code"]

		if err := verifySignatureAndCompare(pubKeyType, pubKeyValue, signedToken); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		// Verification successful
		return c.JSON(fiber.Map{"message": "Signature verified successfully."})
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid process"})
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

func verifySignatureAndCompare(pubkeyType, pubkeyValue, signedToken string) error {
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
		fmt.Println("SUCCEESSSSSSSS")
	}
	return nil
}
