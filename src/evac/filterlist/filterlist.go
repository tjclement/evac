package filterlist

type ABPFilterList struct {
	blacklist partialFilterList
	whitelist partialFilterList
}

func NewABPFilterList(blacklist []Filter, whitelist []Filter) (*ABPFilterList) {
	return &ABPFilterList{blacklist: makePartialList(blacklist), whitelist: makePartialList(whitelist)}
}

func (filterlist *ABPFilterList) Matches(domain string) bool {
	return !filterlist.whitelist.Matches(domain) && filterlist.blacklist.Matches(domain)
}

type partialFilterList struct {
	rules []Filter
}

func makePartialList(filters []Filter) (partialFilterList) {
	filterlist := &partialFilterList{rules: filters}
	return *filterlist
}

func (filterlist *partialFilterList) Matches(domain string) bool {
	for _, v := range filterlist.rules {
		if v.Matches(domain) {
			return true
		}
	}
	return false
}
