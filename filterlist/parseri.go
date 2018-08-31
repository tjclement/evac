package filterlist

import "io"

type FilterParser interface {
	Parse(reader io.Reader) (whitelist []Filter, blacklist []Filter, err error)
}
