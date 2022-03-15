package relayer

import (
	context "context"
	"log"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	grpc "google.golang.org/grpc"
)

const JSONRPC_ETH_BLOCKNUMBER = `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
const JSONRPC_ETH_GETBALANCE = `{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xEA674fdDe714fd979de3EdF0F56AA9716B898ec8", "latest"],"id":77}`

func TestClient(ctx context.Context, addr string, clientCtx client.Context) {
	//
	// Set up a connection to the server.
	log.Println("TestClient connecting to", addr)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := NewRelayerClient(conn)

	//
	//
	relayRequest := &RelayRequest{
		Data: []byte(JSONRPC_ETH_BLOCKNUMBER),
	}
	_, name, _, err := client.GetFromFields(clientCtx.Keyring, clientCtx.From, false)
	if err != nil {
		log.Fatalln("client.GetFromFields", err)
	}

	//
	// Sign
	armor, err := clientCtx.Keyring.ExportPrivKeyArmor(name, "")
	if err != nil {
		log.Fatalln("clientCtx.Keyring.ExportPrivKeyArmor", err)
	}
	log.Println(armor)

	privKey, algo, err := crypto.UnarmorDecryptPrivKey(armor, "")
	if err != nil {
		log.Fatalln("crypto.UnarmorDecryptPrivKey", err)
	}
	log.Println("TODO: check compatible algo", algo)
	priv, _ := btcSecp256k1.PrivKeyFromBytes(btcSecp256k1.S256(), privKey.Bytes())

	//
	// Hash
	ddd := tendermintcrypto.Sha256([]byte(relayRequest.String()))

	// Sign
	//
	sig, err := btcSecp256k1.SignCompact(btcSecp256k1.S256(), priv, ddd, false)
	if err != nil {
		log.Fatalln("btcSecp256k1.SignCompact", err)
	}
	relayRequest.Sig = sig

	key, _ := clientCtx.Keyring.Key(name)
	pub := key.GetPubKey()

	//
	// Recover
	recPub, _, err := btcSecp256k1.RecoverCompact(btcSecp256k1.S256(), sig, ddd)
	if err != nil {
		log.Println("btcSecp256k1.RecoverCompact", err)
	}

	pk := recPub.SerializeCompressed()
	bebe := (secp256k1.PubKey)(pk)

	log.Println(bebe.Address(), pub.Address())

	reply, err := c.Relay(ctx, relayRequest)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(reply)

	//
	reply, err = c.Relay(ctx, &RelayRequest{
		Data: []byte(JSONRPC_ETH_GETBALANCE),
	})
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(reply)
}
