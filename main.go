package main

import (
	"fmt"
	"github.com/miekg/dns"
)

/* TODO: 1 - Read incoming DNS request */
/* TODO: 2 - Check cache for request response */
/* TODO: 3 - Check blacklist for request domain */
/* TODO: 4 - Request from remote DNS server */
/* TODO: 5 - Serve DNS response to client */

func main() {
	m := new(dns.Msg)
	m.SetQuestion("google.com.", dns.TypeA)

	in, _ := dns.Exchange(m, "8.8.8.8:53")
	if t, ok := in.Answer[0].(*dns.A); ok {
		fmt.Printf("Message response: %s\n", t.A)
	}

	fmt.Printf("hello, world\n")
}
