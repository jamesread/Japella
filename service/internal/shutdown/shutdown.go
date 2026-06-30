package shutdown

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	activeServer *http.Server
	stopOnce     sync.Once
)

const (
	responseFlushDelay   = 300 * time.Millisecond
	gracefulShutdownWait = 2 * time.Second
	forcedExitAfter      = 3 * time.Second
)

// RegisterServer records the active HTTP server for graceful shutdown.
func RegisterServer(server *http.Server) {
	activeServer = server
}

// RequestStop shuts down the HTTP server and exits the process after a short delay
// so the API response can be delivered. Container runtimes with a restart policy
// (for example Docker restart: unless-stopped) will start a new instance.
func RequestStop() {
	stopOnce.Do(func() {
		go func() {
			time.Sleep(responseFlushDelay)

			log.Warn("Service stop requested; shutting down")

			// Never block container restart indefinitely on long-lived HTTP/2 or bot connections.
			go func() {
				time.Sleep(forcedExitAfter)
				log.Error("Shutdown timed out; forcing process exit")
				os.Exit(1)
			}()

			if activeServer != nil {
				ctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownWait)
				if err := activeServer.Shutdown(ctx); err != nil {
					log.Warnf("Graceful HTTP shutdown: %v", err)
				}
				cancel()

				if err := activeServer.Close(); err != nil {
					log.Warnf("HTTP server close: %v", err)
				}
			}

			log.Warn("Process exiting")
			os.Exit(0)
		}()
	})
}
