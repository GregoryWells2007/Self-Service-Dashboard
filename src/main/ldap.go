package main

import (
	"crypto/tls"
	"fmt"
	"strings"

	"astraltech.xyz/accountmanager/src/logging"
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

func connectToLDAPServer(URL string, starttls bool, ignore_cert bool) *LDAPServer {
	logging.Debugf("Connecting to LDAP server %s", URL)
	l, err := ldap.DialURL(URL)
	if err != nil {
		logging.Fatal("Failed to connect to LDAP server")
		logging.Fatal(err.Error())
	}
	logging.Infof("Connected to LDAP server")

	if starttls {
		logging.Debugf("Enabling StartTLS")
		if err := l.StartTLS(&tls.Config{InsecureSkipVerify: ignore_cert}); err != nil {
			logging.Errorf("StartTLS failed %s", err.Error())
		}
		logging.Infof("StartTLS enabled")
	}

	return &LDAPServer{
		Connection:         l,
		URL:                URL,
		StartTLS:           starttls,
		IgnoreInsecureCert: ignore_cert,
	}
}

func reconnectToLDAPServer(server *LDAPServer) error {
	logging.Debugf("Reconnecting to %s LDAP server", server.URL)
	if server == nil {
		logging.Errorf("Cannot reconnect: server is nil")
		return fmt.Errorf("Server is nil")
	}

	l, err := ldap.DialURL(server.URL)
	if err != nil {
		logging.Errorf("Failed to connect to LDAP server (has server gone down)")
		return err
	}

	if server.StartTLS {
		logging.Debugf("StartTLS enabling")
		if err := l.StartTLS(&tls.Config{InsecureSkipVerify: server.IgnoreInsecureCert}); err != nil {
			logging.Error("StartTLS failed")
			return err
		}
		logging.Debugf("Successfully Started TLS")
	}

	server.Connection = l
	return nil
}

func connectAsLDAPUser(server *LDAPServer, bindDN, password string) error {
	logging.Debugf("Connecting to %s LDAP server with %s BindDN", server.URL, bindDN)
	if server == nil {
		logging.Errorf("Failed to connect as user, LDAP server is NULL")
		return fmt.Errorf("LDAP server is null")
	}

	if server.Connection == nil || server.Connection.IsClosing() {
		err := reconnectToLDAPServer(server)
		return err
	}
	err := server.Connection.Bind(bindDN, password)
	if err != nil {
		logging.Errorf("Failed to bind to LDAP as user %s", err.Error())
		return err
	}
	return nil
}

func searchLDAPServer(server *LDAPServer, baseDN string, searchFilter string, attributes []string) LDAPSearch {
	logging.Debugf("Searching %s LDAP server\n\tBase DN: %s\n\tSearch Filter %s\n\tAttributes: %s", server.URL, baseDN, searchFilter, strings.Join(attributes, ","))
	if server == nil {
		logging.Errorf("Server is nil, failed to search LDAP server")
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
		logging.Errorf("Failed to search LDAP server %s\n", err.Error())
		return LDAPSearch{false, nil}
	}

	return LDAPSearch{true, sr}
}

func modifyLDAPAttribute(server *LDAPServer, userDN string, attribute string, data []string) error {
	logging.Infof("Modifing LDAP attribute %s", attribute)
	modify := ldap.NewModifyRequest(userDN, nil)
	modify.Replace(attribute, data)
	err := server.Connection.Modify(modify)
	if err != nil {
		logging.Errorf("Failed to modify %s", err.Error())
		return err
	}
	return nil
}

func closeLDAPServer(server *LDAPServer) {
	if server != nil && server.Connection != nil {
		logging.Debug("Closing connection to LDAP server")
		err := server.Connection.Close()
		if err != nil {
			logging.Errorf("Failed to close LDAP server %s", err.Error())
		}
	}
}

func ldapEscapeFilter(input string) string { return ldap.EscapeFilter(input) }
