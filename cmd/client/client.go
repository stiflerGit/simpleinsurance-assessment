package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var ( //flags
	addr      = flag.String("address", "http://localhost:8080", "address of the server")
	frequency = flag.Int("frequency", 1, "number of request each second")
)

func main() {
	flag.Parse()

	period := time.Duration(float64(time.Second) / float64(*frequency))

	ticks := time.Tick(period)

	for range ticks {
		resp, err := http.DefaultClient.Get(*addr)
		if err != nil {
			log.Fatalf("doing get request to %s: %v", *addr, err)
		}

		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("reading response body: %v", err)
		}

		fmt.Printf("server response: %s\n", string(bytes))
	}
}
