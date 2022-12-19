package favicon

import (
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/lavanet/lava/utils"
)

const FaviconFlag = "favicon-path"

// Config defines the config for middleware.
type Config struct {
	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next func(c *fiber.Ctx) bool
	// File holds the path to an actual favicon that will be cached
	//
	// Optional. Default: "Lava icon",
	Data []byte `json:"data"`
	// Optional if you would like to change the favicon file you can provide a path to your favicon file
	FaviconFilePath string `json:"favicon_file_path"`
	// FileSystem is an optional alternate filesystem to search for the favicon in.
	// An example of this could be an embedded or network filesystem
	//
	// Optional. Default: nil
	FileSystem http.FileSystem `json:"-"`
	// CacheControl defines how the Cache-Control header in the response should be set
	//
	// Optional. Default: "public, max-age=31536000"
	CacheControl string `json:"cache_control"`
}

const (
	hType  = "image/x-icon"
	hAllow = "GET, HEAD, OPTIONS"
	hZero  = "0"
)

func GetDefaultParametersConfig(faviconFilePath string) *Config {

	return &Config{
		Next:            nil,
		Data:            lavaIcon,
		FaviconFilePath: faviconFilePath,
		CacheControl:    "public, max-age=31536000",
	}
}

func (cfg *Config) parseFaviconIconFromPath() {
	if cfg.FaviconFilePath == "" {
		return
	}
	data, err := ioutil.ReadFile(cfg.FaviconFilePath)
	if err != nil {
		utils.LavaFormatError("Failed to Read File", err, &map[string]string{"file": cfg.FaviconFilePath})
		return
	}
	cfg.Data = data
}

// New creates a new middleware handler
func New(optionalFaviconPath string) fiber.Handler {
	var cfg *Config

	cfg = GetDefaultParametersConfig(optionalFaviconPath)
	// Load icon if provided
	cfg.parseFaviconIconFromPath()

	// Return new handler
	return func(c *fiber.Ctx) error {
		// Don't execute middleware if Next returns true
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		// Only respond to favicon requests
		if len(c.Path()) != 12 || c.Path() != "/favicon.ico" {
			return c.Next()
		}

		// Only allow GET, HEAD and OPTIONS requests
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			if c.Method() != fiber.MethodOptions {
				c.Status(fiber.StatusMethodNotAllowed)
			} else {
				c.Status(fiber.StatusOK)
			}
			c.Set(fiber.HeaderAllow, hAllow)
			c.Set(fiber.HeaderContentLength, hZero)
			return nil
		}

		// Serve cached favicon
		if len(cfg.Data) > 0 {
			c.Set(fiber.HeaderContentLength, strconv.Itoa(len(cfg.Data)))
			c.Set(fiber.HeaderContentType, hType)
			c.Set(fiber.HeaderCacheControl, cfg.CacheControl)
			return c.Status(fiber.StatusOK).Send(cfg.Data)
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}
