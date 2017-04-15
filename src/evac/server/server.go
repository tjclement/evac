package server

import (
	"fmt"
	"github.com/miekg/dns"
)

type Request struct {
	dns.ResponseWriter
	*dns.Msg
}

type DnsServer struct {
	IncomingRequests chan Request
}

func (server DnsServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	fmt.Print(r.Answer)
	server.IncomingRequests <- Request{w, r}
}

func (server DnsServer) Start(address string) error {
	return dns.ListenAndServe(address, "udp", server)
}