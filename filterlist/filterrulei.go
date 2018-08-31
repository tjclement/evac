package filterlist

type Filter interface {
	Matches(domain string) bool
}