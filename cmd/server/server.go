package main

import (
	"flag"
	"fmt"
	"local/simpleinsurance-assessment/pkg/server"
	"log"
	"net/http"
)

var ( // flags
	port = flag.Int("port", 8080, "port on which start the server")
)

func main() {
	flag.Parse()

	myServer, err := server.New()
	if err != nil {
		log.Fatalf("creating new server: %v", err)
	}

	addr := fmt.Sprintf("localhost:%d", *port)

	log.Printf("starting server on %s", addr)

	log.Println(http.ListenAndServe(addr, myServer))
}
