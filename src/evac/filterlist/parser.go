package filterlist

import (
	"io"
	"bufio"
	"strings"
	"log"
)

type ABPFilterParser struct { }

func NewABPFilterParser() (*ABPFilterParser) {
	return &ABPFilterParser{}
}

func (parser *ABPFilterParser) Parse(reader io.Reader) (whitelist []Filter, blacklist []Filter, err error) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		rule, isException := parser.isExceptionRule(scanner.Text())
		if rule, domainRule := parser.isSimpleDomainRule(rule); domainRule {
			filter, err := NewABPFilter(rule)

			if err {
				log.Fatalf("Could not create ABPFilter on rule %s. Error %t", rule, err)
				break
			}

			if isException {
				whitelist = append(whitelist, *filter)
			} else {
				blacklist = append(blacklist, *filter)
			}
		}
	}

	return whitelist, blacklist, err
}

const (
	exceptionPrefix = "@@"
	domainPrefix = "||"
	domainSuffix = "^"
)

func (parser *ABPFilterParser) isSimpleDomainRule(rule string) (string, bool) {
	if cleanedRule, matchesPrefix := parser.checkRulePrefixAndRemove(rule, domainPrefix); matchesPrefix {
		return parser.checkRuleSuffixAndRemove(cleanedRule, domainSuffix)
	}
	return rule, false
}

func (parser *ABPFilterParser) isExceptionRule(rule string) (string, bool) {

	return parser.checkRulePrefixAndRemove(rule, exceptionPrefix)
}

func (*ABPFilterParser) checkRulePrefixAndRemove(rule string, prefix string) (string, bool) {
	if strings.HasPrefix(rule, prefix) {
		return strings.TrimPrefix(rule, prefix), true
	}
	return rule, false
}

func (*ABPFilterParser) checkRuleSuffixAndRemove(rule string, suffix string) (string, bool) {
	if strings.HasSuffix(rule, suffix) {
		return strings.TrimSuffix(rule, suffix), true
	}
	return rule, false
}