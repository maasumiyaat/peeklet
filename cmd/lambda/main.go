// Command lambda is the Function URL entrypoint for Peeklet.
//
// It stays intentionally tiny: load config, construct the API server (which
// builds its own AWS clients and sub-services), and start the Lambda. All
// routing and handler logic lives in internal/api, so adding handlers in later
// chunks never requires touching this file.
package main

import (
	"context"
	"log"

	"peeklet/internal/api"
	"peeklet/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	srv, err := api.NewServer(context.Background(), cfg)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	srv.Start()
}
