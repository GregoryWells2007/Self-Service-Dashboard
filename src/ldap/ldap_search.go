package ldap

import (
	"github.com/go-ldap/ldap/v3"
)

type LDAPSearch struct {
	search *ldap.SearchResult
}

type LDAPEntry struct {
	entry *ldap.Entry
}

func (s *LDAPSearch) EntryCount() int {
	return len(s.search.Entries)
}

func (s *LDAPSearch) GetEntry(number int) *LDAPEntry {
	return &LDAPEntry{s.search.Entries[number]}
}

func (e *LDAPEntry) GetRawAttributeValue(name string) []byte {
	return e.entry.GetRawAttributeValue(name)
}
