package ldap

import (
	"github.com/go-ldap/ldap/v3"
)

func LDAPEscapeFilter(input string) string { return ldap.EscapeFilter(input) }
