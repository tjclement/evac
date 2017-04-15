package server

import (
	"github.com/miekg/dns"
)

type Request struct {
	dns.ResponseWriter
	*dns.Msg
}

type DnsServer struct {
	IncomingRequests chan Request
}

func NewServer() (*DnsServer) {
	return &DnsServer{make(chan Request)}
}

func (server DnsServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	server.IncomingRequests <- Request{w, r}
}

func (server DnsServer) Start(address string) error {
	return dns.ListenAndServe(address, "udp", server)
}