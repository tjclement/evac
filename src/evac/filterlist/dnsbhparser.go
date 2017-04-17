package filterlist

import (
	"io"
	"bufio"
	"log"
)

type DNSBHFilterParser struct {

}

func NewDNSBHFilterParser() (*DNSBHFilterParser) {
	return &DNSBHFilterParser{}
}

func (parser *DNSBHFilterParser) Parse(reader io.Reader) (whitelist []Filter, blacklist []Filter, err error) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		var compiled *RegexFilter

		rule := scanner.Text()

		compiled, err = NewRegexFilter(rule)
		if err {
			log.Panicf("Could not create RegexFilter on rule %s. Error %t", rule, err)
			break
		}

		whitelist = append(whitelist, *compiled)
	}

	return whitelist, blacklist, err
}