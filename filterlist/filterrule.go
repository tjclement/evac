package filterlist

import "regexp"

type RegexFilter struct {
	rule_expression *regexp.Regexp
}

func NewRegexFilter(rule_regex string) (*RegexFilter, error) {
	compiled_regex, err := regexp.Compile(rule_regex)
	return &RegexFilter{rule_expression: compiled_regex}, err
}


func (rule *RegexFilter) Matches(domain string) bool {
	return rule.rule_expression.MatchString(domain)
}