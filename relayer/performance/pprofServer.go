package performance

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	fiberpprof "github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/lavanet/lava/utils"
)

const (
	EnablePprofFlagName      = "enable-pprof"
	PprofAddressFlagName     = "pprof-address"
	PprofAddressDefaultValue = "127.0.0.1:8080"
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

	utils.LavaFormatInfo("start pprof HTTP server", &map[string]string{"IPAddress": addr})

	return nil
}
