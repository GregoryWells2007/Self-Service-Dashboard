package main

import (
	"crypto/tls"
	"errors"
	"log"

	"github.com/go-ldap/ldap/v3"
)

type LDAPServer struct {
	URL                string
	StartTLS           bool
	IgnoreInsecureCert bool
	Connection         *ldap.Conn
}

type LDAPSearch struct {
	Succeeded  bool
	LDAPSearch *ldap.SearchResult
}

func connectToLDAPServer(URL string, starttls bool, ignore_cert bool) (*LDAPServer, error) {
	l, err := ldap.DialURL(URL)
	if err != nil {
		return nil, err
	}

	if starttls {
		if err := l.StartTLS(&tls.Config{InsecureSkipVerify: ignore_cert}); err != nil {
			log.Println("StartTLS failed:", err)
		}
	}

	return &LDAPServer{
		Connection:         l,
		URL:                URL,
		StartTLS:           starttls,
		IgnoreInsecureCert: ignore_cert,
	}, nil
}

func reconnectToLDAPServer(server *LDAPServer) {
	if server == nil {
		log.Println("Cannot reconnect: server is nil")
		return
	}

	l, err := ldap.DialURL(server.URL)
	if err != nil {
		log.Print(err)
		return
	}

	if server.StartTLS {
		if err := l.StartTLS(&tls.Config{InsecureSkipVerify: server.IgnoreInsecureCert}); err != nil {
			log.Println("StartTLS failed:", err)
		}
	}

	server.Connection = l
}

func connectAsLDAPUser(server *LDAPServer, bindDN, password string) error {
	if server == nil {
		return errors.New("LDAP server is nil")
	}

	// Reconnect if needed
	if server.Connection == nil || server.Connection.IsClosing() {
		reconnectToLDAPServer(server)
	}
	return server.Connection.Bind(bindDN, password)
}

func searchLDAPServer(server *LDAPServer, baseDN string, searchFilter string, attributes []string) LDAPSearch {
	if server == nil {
		return LDAPSearch{false, nil}
	}

	if server.Connection == nil {
		reconnectToLDAPServer(server)
		if server.Connection == nil {
			return LDAPSearch{false, nil}
		}
	}

	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		searchFilter, attributes,
		nil,
	)

	sr, err := server.Connection.Search(searchRequest)
	if err != nil {
		return LDAPSearch{false, nil}
	}

	return LDAPSearch{true, sr}
}

func modifyLDAPAttribute(server *LDAPServer, userDN string, attribute string, data []string) error {
	modify := ldap.NewModifyRequest(userDN, nil)
	modify.Replace(attribute, data)
	err := server.Connection.Modify(modify)
	if err != nil {
		return err
	}
	return nil
}

func closeLDAPServer(server *LDAPServer) {
	if server != nil && server.Connection != nil {
		server.Connection.Close()
	}
}

func ldapEscapeFilter(input string) string { return ldap.EscapeFilter(input) }
