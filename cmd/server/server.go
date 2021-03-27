package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/stiflerGit/simpleinsurance-assessment/pkg/server"
)

var ( // flags
	port            = flag.Int("port", 8080, "port on which start the server")
	persistenceFile = flag.String("persistence", "", "path of the file to read/write state")
)

func main() {
	flag.Parse()

	serverOpts := []server.Option{
		server.WithLogger(log.New(os.Stderr, "server", log.LstdFlags|log.Lshortfile)),
	}
	if *persistenceFile != "" {
		serverOpts = append(serverOpts, server.WithFilePersistencePath(*persistenceFile))
	}

	myServer, err := server.New(serverOpts...)
	if err != nil {
		log.Fatalf("creating new server: %v", err)
	}

	ctx, cancelFunch := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFunch()

	if err := myServer.Start(ctx); err != nil {
		log.Fatalf("[ERROR] starting the server: %v", err)
	}

	addr := fmt.Sprintf("localhost:%d", *port)

	log.Printf("starting server on %s", addr)

	log.Println(http.ListenAndServe(addr, myServer))
}
