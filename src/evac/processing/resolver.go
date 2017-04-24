package processing

import (
	"github.com/miekg/dns"
	"time"
	"sync/atomic"
	"errors"
	"sync"
)

type Resolver struct {
	cache *Cache
	ttlTimeCheck time.Duration
	running atomic.Value
	client *dns.Client
	externalDNS string
	ttlCheckSem sync.WaitGroup
}

func NewResolver(cacheCapacity uint32, externalDNS string, timeToTTLCheck time.Duration) (*Resolver) {
	resolver := &Resolver{cache: NewCache(cacheCapacity), client: new(dns.Client), ttlTimeCheck: timeToTTLCheck, externalDNS: externalDNS}
	resolver.running.Store(true)
	resolver.ttlCheckSem.Add(1)
	go resolver.ttlExpirationCheck()
	return resolver
}

func (resolver *Resolver) Close() {
	resolver.running.Store(false)
	resolver.ttlCheckSem.Wait()
}

func (resolver *Resolver) Resolve(questions []dns.Question) ([]dns.RR, error) {
	isRunning := resolver.running.Load().(bool)
	if !isRunning {
		return nil, errors.New("Cannot process request; resolver is not running.")
	}
	results, err := resolver.answerMessage(questions)
	return results, err
}

func (resolver *Resolver) answerMessage(questions []dns.Question) ([]dns.RR, error) {
	toResolve := make([]dns.Question, 1)
	resolved := make([]dns.RR, 1)
	for _, question := range questions {
		record, ok := resolver.cache.GetRecord(question.Name, question.Qtype)
		if !ok {
			toResolve = append(toResolve, question)
			continue
		}
		resolved = append(resolved, record)
	}
	resolvedExternal, err := resolver.resolveExternal(toResolve)

	if err != nil {
		return nil, err
	}

	resolved = append(resolved, resolvedExternal...)
	return resolved, nil
}

func (resolver *Resolver) resolveExternal(questions []dns.Question) ([]dns.RR, error) {
	resolved := make([]dns.RR, 1)
	message := new(dns.Msg)

	for _, question := range questions {
		message.SetQuestion(question.Name, question.Qtype)
	}

	reply, _, err := resolver.client.Exchange(message, resolver.externalDNS)
	if err != nil {
		return nil, err
	}

	for _, answer := range reply.Answer {
		resolver.cache.UpdateRecord(answer.Header().Name, answer)
		resolved = append(resolved, answer)
	}

	return resolved, nil
}

func (resolver *Resolver) ttlExpirationCheck() {
	for true {
		running := resolver.running.Load().(bool)
		if !running {
			break
		}
		time.Sleep(resolver.ttlTimeCheck)
		resolver.cache.TTLExpirationCleanup()
	}
	resolver.ttlCheckSem.Done()
}