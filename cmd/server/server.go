package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/stiflerGit/simpleinsurance-assessment/pkg/server"
)

var ( // flags
	port            = flag.Int("port", 8080, "port on which start the server")
	persistenceFile = flag.String("persistence", "", "path of the file to read/write state")
)

func main() {
	flag.Parse()

	var serverOpts []server.Option
	if *persistenceFile != "" {
		serverOpts = append(serverOpts, server.WithFilePersistence(*persistenceFile))
	}

	myServer, err := server.New(serverOpts...)
	if err != nil {
		log.Fatalf("creating new server: %v", err)
	}

	addr := fmt.Sprintf("localhost:%d", *port)

	log.Printf("starting server on %s", addr)

	log.Println(http.ListenAndServe(addr, myServer))
}
