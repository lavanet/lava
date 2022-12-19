package performance

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/lavanet/lava/utils"
)

const EnablePprofFlagName = "enable-pprof"
const PprofAddressFlagName = "pprof-address"
const PprofAddressDefaultValue = "127.0.0.1:8080"

func StartPprofServer(addr string) error {
	// validate the input
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return utils.LavaFormatError("invalid pprof server address input", err, nil)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return utils.LavaFormatError("invalid pprof server IP address", err, nil)
	}
	_, err = strconv.ParseInt(port, 10, 16)
	if err != nil {
		return utils.LavaFormatError("invalid pprof server port", err, nil)
	}

	// create an error channel for the HTTP server
	errCh := make(chan error)

	// start an HTTP server as a goroutine
	go func() {
		serverAddress := fmt.Sprintf("%s:%s", ip, port)
		l, err := net.Listen("tcp", serverAddress)
		if err != nil {
			errCh <- fmt.Errorf("pprof http server init failed. port is taken. err: %v", err)
			return
		}
		err = http.Serve(l, nil)
		if err != nil {
			errCh <- fmt.Errorf("pprof http server failed. err: %v", err)
		}

	}()

	// Use the select statement to check for an error on the channel without blocking TODO: errors not handled for some reason
	select {
	case err := <-errCh:
		if err != nil {
			return utils.LavaFormatError("pprof http server error", err, nil)
		}
	default:
		// do nothing
	}

	utils.LavaFormatInfo("start pprof HTTP server", &map[string]string{"IPAddress": host, "port": port})

	return nil
}
