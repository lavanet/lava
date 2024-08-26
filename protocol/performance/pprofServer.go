package performance

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	fiberpprof "github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/lavanet/lava/v2/utils"
)

const (
	PprofAddressFlagName = "pprof-address"
)

func StartPprofServer(addr string) error {
	// Set up the Fiber app
	app := fiber.New()

	// Let the fiber HTTP server use pprof
	app.Use(fiberpprof.New())

	// Start the HTTP server in a goroutine
	go func() {
		fmt.Printf("Starting server on %s\n", addr)
		if err := app.Listen(addr); err != nil {
			fmt.Printf("Error starting pprof HTTP server: %s\n", err)
		}
	}()

	utils.LavaFormatInfo("start pprof HTTP server", utils.Attribute{Key: "IPAddress", Value: addr})

	return nil
}
