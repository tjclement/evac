package filterlist

import "regexp"

type ABPFilter struct {
	rule_expression *regexp.Regexp
}

func NewABPFilter(rule_regex string) (*ABPFilter, error) {
	compiled_regex, err := regexp.Compile(rule_regex)
	return &ABPFilter{rule_expression: compiled_regex}, err
}


func (rule * ABPFilter) Matches(domain string) bool {
	return rule.rule_expression.MatchString(domain)
}