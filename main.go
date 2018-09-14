package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/vlabakje/evac/filterlist"
	"github.com/vlabakje/evac/processing"
	"github.com/vlabakje/evac/server"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	port := flag.Int("port", 53, "Port to run on")
	cache_size := flag.Uint("cache", 200, "The amount of DNS responses to cache locally to increase performance")
	recursion_address := flag.String("recursion_address", "8.8.8.8:53", "Server address in the format of 'ip:port' to query non-cached requests from")
	worker_amount := flag.Uint("worker_amount", 5, "The amount of workers that concurrently accept DNS requests")
	abp_filter := flag.String("abp_filter", "https://pgl.yoyo.org/as/serverlist.php?hostformat=adblockplus&showintro=0&mimetype=plaintext", "The AdBlockPlus formatted list of domains to process for blacklisting")
	flag.Parse()

	cache := processing.NewCache(uint32(*cache_size))
	initialFilterDownloadFailed := false
	filter, err := loadFilter(abp_filter)
	if err != nil {
		fmt.Printf("%s, unable to download filter, starting without one and initiating period retries\n", err)
		filter = filterlist.NewABPFilterList(nil, nil)
		initialFilterDownloadFailed = true
	}

	fmt.Printf("Starting server on port %d\r\n", *port)
	listener := server.NewServer(cache, filter, recursion_address, uint16(*worker_amount))
	go listener.Start(fmt.Sprintf(":%d", *port))

	go func() {
		var input string
		scanner := bufio.NewScanner(os.Stdin)
		commandFuncs := map[string]func(*server.DnsServer, []string, *string){
			"help":   printHelp,
			"snoop":  toggleSnoop,
			"reload": reloadFilter,
		}

		fmt.Print("> ")
		for scanner.Scan() {
			input = scanner.Text()
			pieces := strings.Split(input, " ")
			command := pieces[0]

			function, exists := commandFuncs[command]
			if exists {
				function(listener, pieces, abp_filter)
			} else {
				printHelp(listener, pieces, abp_filter)
			}

			fmt.Print("> ")
		}
	}()

	ticker := time.NewTicker(24 * time.Hour)
	if initialFilterDownloadFailed {
		ticker.Stop()
		ticker = time.NewTicker(2 * time.Minute)
	}
	go func() {
		for {
			select {
			case <- ticker.C:
				fmt.Printf("\nperiodic reload of filter:\n")
				filter, err := loadFilter(abp_filter)
				if err != nil {
					fmt.Printf("%s, unable to reload\n", err)
				} else {
					listener.ReloadFilter(filter)
					if initialFilterDownloadFailed {
						initialFilterDownloadFailed = false
						ticker.Stop()
						ticker = time.NewTicker(24 * time.Hour)
					}
				}
			}
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func printHelp(dnsServer *server.DnsServer, params []string, abp_filter *string) {
	fmt.Println("Available commands:")
	fmt.Println("snoop [ip] - Enable logging of requests, for all IPs or only the one specified")
	fmt.Println("reload - Reload the ABP rulelist")
	fmt.Println("help - Show this help message")
}

func toggleSnoop(dnsServer *server.DnsServer, params []string, abp_filter *string) {
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

func reloadFilter(dnsServer *server.DnsServer, params []string, abp_filter *string) {
	filter, err := loadFilter(abp_filter)
	if err != nil {
		fmt.Printf("%s, unable to reload\n", err)
	} else {
		dnsServer.ReloadFilter(filter)
	}
}

func loadFilter(abp_filter *string) (filter filterlist.Filter, err error) {
	resp, err := http.Get(*abp_filter)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error opening AdBlockPlus filter list '%s'", *abp_filter))
	}

	parser := filterlist.NewABPFilterParser()
	blacklist, whitelist, err := parser.Parse(resp.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error parsing AdBlockPlus filter list"))
	}
	defer resp.Body.Close()

	filter = filterlist.NewABPFilterList(blacklist, whitelist)
	fmt.Printf("filter loaded with %d blacklists, %d whitelists\n", len(blacklist), len(whitelist))
	return filter, nil
}
