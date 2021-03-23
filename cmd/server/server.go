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

	myServer := server.New()

	addr := fmt.Sprintf(":%d", *port)
	log.Println(http.ListenAndServe(addr, myServer))
}
