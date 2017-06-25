package main

import (
	"fmt"
	"flag"
	"evac/server"
	"evac/processing"
)
func main() {
	port := flag.Int("port", 1053, "Port to run on")
	flag.Parse()

	cache := processing.NewCache(200)
	listener := server.NewServer(50, cache)
	listener.Start(fmt.Sprintf(":%d", *port))
}
