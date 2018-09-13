package server

import (
	"fmt"
	"github.com/miekg/dns"
	"github.com/vlabakje/evac/filterlist"
	"github.com/vlabakje/evac/processing"
	"strings"
	"time"
)

type Request struct {
	Response dns.ResponseWriter
	Message  *dns.Msg
}

type DnsServer struct {
	IncomingRequests chan Request
	ShouldPrint      *bool
	IPFilter         *string
	cache            *processing.Cache
	filter           filterlist.Filter
	recursionAddress *string
	workerAmount     uint16
}

func NewServer(cache *processing.Cache, filter filterlist.Filter, recursion_address *string, worker_amount uint16) *DnsServer {
	shouldPrint := false
	ipFilter := ""
	return &DnsServer{make(chan Request, worker_amount*10), &shouldPrint, &ipFilter, cache, filter, recursion_address, worker_amount}
}

func (server DnsServer) ServeDNS(writer dns.ResponseWriter, request *dns.Msg) {
	/* Request is handled by a worker with processRequest() */
	server.IncomingRequests <- Request{writer, request}
}

func (server DnsServer) Start(address string) error {
	/* Start configured amount of workers that accept requests from the IncomingRequests channel */
	for i := uint16(0); i < server.workerAmount; i++ {
		go server.acceptRequests()
	}

	/* Listen for DNS requests */
	err := dns.ListenAndServe(address, "udp", server)

	if err != nil {
		fmt.Printf("Error setting up listening socket: %s\r\n", err.Error())
		return err
	}

	return nil
}

/* Forwards a DNS request to an external DNS server, and returns its result. */
func (server DnsServer) recurse(question dns.Question) (*dns.Msg, time.Duration, error) {
	c := new(dns.Client)
	m := new(dns.Msg)
	m.Id = dns.Id()
	m.RecursionDesired = true
	m.Question = append(m.Question, question)
	return c.Exchange(m, *server.recursionAddress)
}

func (server DnsServer) acceptRequests() {
	for true {
		request := <-server.IncomingRequests
		server.processRequest(request.Response, request.Message)
	}
}

func (server DnsServer) processRequest(writer dns.ResponseWriter, request *dns.Msg) error {
	response := new(dns.Msg)
	response.SetReply(request)

	if len(request.Question) != 1 {
		response.Rcode = dns.RcodeFormatError
		writer.WriteMsg(response)
		return server.writeResponse(writer, response, "No question")
	}

	/* DNS RFC supports multiple questions, but in practise no DNS servers do. E.g. response status code NXDOMAIN
	 * does not make sense if there is more than one question, so in reality there is always only one. */
	question := request.Question[0]

	/* Check if the question is in our local cache, and if so, immediately return it. */
	records, exists, is_blocked := server.cache.GetRecord(question.Name, question.Qtype)
	if exists {
		if is_blocked {
			response.Rcode = dns.RcodeNameError
		} else {
			response.Answer = records
		}
		return server.writeResponse(writer, response, "Cached")
	}

	/* Check if question is in blacklist. */
	if server.filter.Matches(question.Name) {
		response.Rcode = dns.RcodeNameError
		server.cache.UpdateBlockedRecord(question.Name, question.Qtype)
		return server.writeResponse(writer, response, "Blacklisted")
	}

	/* Forward unresolved question to another server */
	recursion_response, _, err := server.recurse(question)

	if err != nil || recursion_response == nil {
		response.Rcode = dns.RcodeServerFailure
		return server.writeResponse(writer, response, "Forward failure")
	}

	if len(recursion_response.Answer) >= 1 {
		server.cache.UpdateRecord(question.Name, question.Qtype, recursion_response.Answer)
		response.Answer = recursion_response.Answer
	}

	return server.writeResponse(writer, response, "Forwarded")
}

func (server *DnsServer) writeResponse(writer dns.ResponseWriter, response *dns.Msg, logPreface string) error {
	fromAddress := writer.RemoteAddr().String()
	fromIP := strings.Split(fromAddress, ":")[0]

	response.RecursionAvailable = true

	if *server.ShouldPrint && (len(*server.IPFilter) == 0 || *server.IPFilter == fromIP) {
		fmt.Println("\r\n", logPreface)
		for _, question := range response.Question {
			fmt.Printf("Question: %s - Qtype %d Qclass %d\r\n", question.Name, question.Qtype, question.Qclass)
		}
		for _, question := range response.Answer {
			fmt.Printf("Answer: %s\r\n", question.String())
		}
		fmt.Println("--------------")
	}

	return writer.WriteMsg(response)
}

func (server *DnsServer) ReloadFilter(newfilter filterlist.Filter) {
	server.filter = newfilter
	server.cache.Flush()
}
