package main

import (
	"fmt"
	"flag"
	"evac/server"
	"evac/filterlist"
	"evac/processing"
	"os"
)
func main() {
	port := flag.Int("port", 53, "Port to run on")
	cache_size := flag.Uint("cache", 200, "The amount of DNS responses to cache locally to increase performance")
	recursion_address := flag.String("recursion_address", "8.8.8.8:53", "Server address in the format of 'ip:port' to query non-cached requests from")
	flag.Parse()

	cache := processing.NewCache(uint32(*cache_size))
	parser := filterlist.NewABPFilterParser()
	abp_file, err := os.Open("./abp_filter.txt")
	if err != nil {
		fmt.Printf("Error opening AdBlockPlus filter list 'abp_filter.txt', exiting")
		return
	}

	blacklist, whitelist, err := parser.Parse(abp_file)
	if err != nil {
		fmt.Printf("Error parsing AdBlockPlus filter list, exiting")
		return
	}

	fmt.Printf("Starting server on port %d\r\n", *port)
	filter := filterlist.NewABPFilterList(blacklist, whitelist)
	listener := server.NewServer(50, cache, filter, recursion_address)
	listener.Start(fmt.Sprintf(":%d", *port))
}
