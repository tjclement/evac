package server

import (
	"github.com/miekg/dns"
	"evac/processing"
	"evac/filterlist"
	"time"
)

type Request struct {
	Response dns.ResponseWriter
	Message  *dns.Msg
}

type DnsServer struct {
	IncomingRequests  chan Request
	cache             *processing.Cache
	filter            filterlist.Filter
	recursion_address *string
}

func NewServer(queue_size int, cache *processing.Cache, filter filterlist.Filter, recursion_address *string) (*DnsServer) {
	return &DnsServer{make(chan Request, queue_size), cache, filter, recursion_address}
}

func (server DnsServer) ServeDNS(writer dns.ResponseWriter, request *dns.Msg) {
	go(func(writer dns.ResponseWriter, request *dns.Msg) error {
		response := new(dns.Msg)
		response.SetReply(request)

		if len(request.Question) != 1 {
			response.Rcode = dns.RcodeFormatError
			writer.WriteMsg(response)
			return writer.WriteMsg(response)
		}

		/* DNS RFC supports multiple questions, but in practise no DNS servers do. E.g. response status code NXDOMAIN
		 * does not make sense if there is more than one question, so in reality there is always only one. */
		question := request.Question[0]

		/* Check if the question is in our local cache, and if so, immediately return it. */
		records, exists := server.cache.GetRecord(question.Name, question.Qtype)
		if exists {
			response.Answer = records
			return writer.WriteMsg(response)
		}

		/* Check if question is in blacklist. */
		if server.filter.Matches(question.Name) {
			response.Rcode = dns.RcodeNameError
			return writer.WriteMsg(response)
		}

		/* Forward unresolved question to another server */
		recursion_response, _, err := server.Recurse(question)

		if err != nil && &recursion_response == nil {
			response.Rcode = dns.RcodeServerFailure
			return writer.WriteMsg(response)
		}

		if len(recursion_response.Answer) >= 1 {
			server.cache.UpdateRecord(question.Name, recursion_response.Answer)
			response.Answer = recursion_response.Answer
		}

		return writer.WriteMsg(response)
	})(writer, request)
}

/* Forwards a DNS request to an external DNS server, and returns its result. */
func (server DnsServer) Recurse(question dns.Question) (*dns.Msg, time.Duration, error) {
	c := new(dns.Client)
	m := new(dns.Msg)
	m.Id = dns.Id()
	m.RecursionDesired = true
	m.Question = append(m.Question, question)
	return c.Exchange(m, *server.recursion_address)
}

func (server DnsServer) Start(address string) error {
	return dns.ListenAndServe(address, "udp", server)
}
