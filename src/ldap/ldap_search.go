package ldap

import (
	"github.com/go-ldap/ldap/v3"
)

type LDAPSearch struct {
	search *ldap.SearchResult
}

func (s *LDAPSearch) EntryCount() int {
	return len(s.search.Entries)
}
