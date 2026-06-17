package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/4current/relayops/internal/agent"
)

func main() {
	listen := flag.String("listen", ":8787", "listen address")
	flag.Parse()

	ctx := context.Background()

	srv := agent.NewServer(ctx)

	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("relayops-agent listening on %s", *listen)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
