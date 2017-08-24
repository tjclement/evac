package main

import (
	"fmt"
	"flag"
	"evac/server"
	"evac/filterlist"
	"evac/processing"
	"os"
	"os/signal"
	"strings"
	"bufio"
)
func main() {
	port := flag.Int("port", 53, "Port to run on")
	cache_size := flag.Uint("cache", 200, "The amount of DNS responses to cache locally to increase performance")
	recursion_address := flag.String("recursion_address", "8.8.8.8:53", "Server address in the format of 'ip:port' to query non-cached requests from")
	worker_amount := flag.Uint("worker_amount", 5, "The amount of workers that concurrently accept DNS requests")
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
	listener := server.NewServer(cache, filter, recursion_address, uint16(*worker_amount))
	go listener.Start(fmt.Sprintf(":%d", *port))

	go func(){
		var input string
		scanner := bufio.NewScanner(os.Stdin)
		commandFuncs := map[string]func(*server.DnsServer, []string){
			"help": printHelp,
			"snoop": toggleSnoop,
		}

		fmt.Print("> ")
		for scanner.Scan() {
			input = scanner.Text()
			pieces := strings.Split(input, " ")
			command := pieces[0]

			function, exists := commandFuncs[command]
			if exists {
				function(listener, pieces)
			} else {
				printHelp(listener, pieces)
			}

			fmt.Print("> ")
		}
	}()
	
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<- c
}

func printHelp(dnsServer *server.DnsServer, params []string) {
	fmt.Println("Available commands:")
	fmt.Println("snoop [ip] - Enable logging of requests, for all IPs or only the one specified")
	fmt.Println("help - Show this help message")
}

func toggleSnoop(dnsServer *server.DnsServer, params []string) {
	ipFilter := ""
	if len(params) > 1 {
		ipFilter = params[1]
	}
	*dnsServer.ShouldPrint = !*dnsServer.ShouldPrint
	*dnsServer.IPFilter = ipFilter

	if *dnsServer.ShouldPrint {
		if len(ipFilter) > 0 {
			fmt.Printf("Snooping enabled for %s\r\n", ipFilter)
		} else {
			fmt.Println("Snooping enabled")
		}
	} else {
		fmt.Println("Snooping disabled")
	}
}