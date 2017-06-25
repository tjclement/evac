package server

import (
	"github.com/miekg/dns"
	"evac/processing"
)

type Request struct {
	Response dns.ResponseWriter
	Message *dns.Msg
}

type DnsServer struct {
	IncomingRequests chan Request
	cache *processing.Cache
}

func NewServer(queue_size int, cache *processing.Cache) (*DnsServer) {
	return &DnsServer{make(chan Request, queue_size), cache}
}

func (server DnsServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	go func(request Request) {
		response := new(dns.Msg)
		response.SetReply(request.Message)
		cache_misses := make([]dns.Question, 0)
		recursion_questions := make([]dns.Question, 0)

		/* Check request's questions in local cache */
		for _, question := range request.Message.Question {
			record, exists := server.cache.GetRecord(question.Name, question.Qtype)
			if !exists {
				cache_misses = append(cache_misses, question)
			} else {
				response.Answer = append(response.Answer, record)
			}
		}

		/* Check unresolved questions in blacklist */
		for _, question := range cache_misses {
			/* TODO: 3 - Check blacklist for request domain */
			if question.Name == "doubleclick.net." {
				answ, _ := dns.NewRR(question.Name + " 60 IN A 0.0.0.0")
				response.Answer = append(response.Answer, answ)
				server.cache.UpdateRecord(question.Name, answ)
			} else {
				/* TODO: remove following debug statement when actual blacklist is done */
				recursion_questions = append(recursion_questions, question)
			}
		}

		/* Forward any unresolved questions to another server */
		if len(recursion_questions) > 0 {
			c := new(dns.Client)
			m := new(dns.Msg)
			m.Id = dns.Id()
			m.RecursionDesired = true
			m.Question = recursion_questions

			recursion_response, _, _ := c.Exchange(m, "8.8.8.8:53")
			for index, answer := range recursion_response.Answer {
				response.Answer = append(response.Answer, answer)

				if len(recursion_questions) > index {
					server.cache.UpdateRecord(recursion_questions[index].Name, answer)
				}
			}
		}

		request.Response.WriteMsg(response)
	}(Request{w, r})
}

func (server DnsServer) Start(address string) error {
	return dns.ListenAndServe(address, "udp", server)
}