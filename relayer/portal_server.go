package relayer

import (
	context "context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/lavanet/lava/x/spec/types"
	grpc "google.golang.org/grpc"
)

var g_request_lock sync.Mutex

func PortalServer(
	ctx context.Context,
	clientCtx client.Context,
	queryClient types.QueryClient,
	listenAddr string,
	relayerUrl string,
	specId int,
) {
	//
	// Get specs
	_, clientApis, err := getSpec(ctx, queryClient, specId)
	if err != nil {
		log.Fatalln("error: getSpec", err)
	}
	log.Println(clientApis, specId)
	g_clientApis = clientApis

	//
	// Start sentry
	sentry := NewSentry(clientCtx.Client)
	err = sentry.Init(ctx)
	if err != nil {
		log.Fatalln("error sentry.Init", err)
	}
	go sentry.Start()
	for sentry.GetBlockHeight() == 0 {
		time.Sleep(1 * time.Second)
	}

	//
	// Set up a connection to the server.
	log.Println("PortalServer connecting to", relayerUrl)
	keyName, err := getKeyName(clientCtx)
	if err != nil {
		log.Fatalln("error: getKeyName", err)
	}
	privKey, err := getPrivKey(clientCtx, keyName)
	if err != nil {
		log.Fatalln("error: getPrivKey", err)
	}
	clientKey, _ := clientCtx.Keyring.Key(keyName)
	log.Println("Client pubkey", clientKey.GetPubKey().Address())

	connectCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(connectCtx, relayerUrl, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	relayerClient := NewRelayerClient(conn)
	sessionId := rand.Int63()

	//
	// Setup HTTP Server
	app := fiber.New(fiber.Config{})

	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		var (
			mt  int
			msg []byte
			err error
		)
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				log.Println("read:", err)
				break
			}
			log.Println("in <<< ", string(msg))

			g_request_lock.Lock()
			reply, _, err := sendRelay(ctx, clientCtx, relayerClient, privKey, specId, sessionId, string(msg), sentry.GetBlockHeight())
			g_request_lock.Unlock()
			if err != nil {
				log.Println(err)
				break
			}

			if err = c.WriteMessage(mt, reply.Data); err != nil {
				log.Println("write:", err)
				break
			}
			log.Println("out >>> ", string(reply.Data))
		}
	}))

	app.Post("/", func(c *fiber.Ctx) error {
		g_request_lock.Lock()
		defer g_request_lock.Unlock()

		log.Println("in <<< ", string(c.Body()))
		reply, _, err := sendRelay(ctx, clientCtx, relayerClient, privKey, specId, sessionId, string(c.Body()), sentry.GetBlockHeight())
		if err != nil {
			log.Println(err)
			return nil
		}

		log.Println("out >>> ", string(reply.Data))
		return c.SendString(string(reply.Data))
	})

	//
	// Go
	err = app.Listen(listenAddr)
	if err != nil {
		log.Println(err)
	}
}
