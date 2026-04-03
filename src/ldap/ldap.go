package ldap

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

func ConnectToLDAPServer(URL string, starttls bool, ignore_cert bool) *LDAPServer {
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

func ReconnectToLDAPServer(server *LDAPServer) error {
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

func ConnectAsLDAPUser(server *LDAPServer, bindDN, password string) error {
	logging.Debugf("Connecting to %s LDAP server with %s BindDN", server.URL, bindDN)
	if server == nil {
		logging.Errorf("Failed to connect as user, LDAP server is NULL")
		return fmt.Errorf("LDAP server is null")
	}

	if server.Connection == nil || server.Connection.IsClosing() {
		err := ReconnectToLDAPServer(server)
		return err
	}
	err := server.Connection.Bind(bindDN, password)
	if err != nil {
		logging.Errorf("Failed to bind to LDAP as user %s", err.Error())
		return err
	}
	return nil
}

func SearchLDAPServer(server *LDAPServer, baseDN string, searchFilter string, attributes []string) LDAPSearch {
	logging.Debugf("Searching %s LDAP server\n\tBase DN: %s\n\tSearch Filter %s\n\tAttributes: %s", server.URL, baseDN, searchFilter, strings.Join(attributes, ","))
	if server == nil {
		logging.Errorf("Server is nil, failed to search LDAP server")
		return LDAPSearch{false, nil}
	}

	if server.Connection == nil {
		ReconnectToLDAPServer(server)
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

func ModifyLDAPAttribute(server *LDAPServer, userDN string, attribute string, data []string) error {
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

func ChangeLDAPPassword(server *LDAPServer, userDN, oldPassword, newPassword string) error {
	logging.Infof("Changing LDAP password for %s", userDN)

	if server == nil || server.Connection == nil {
		return fmt.Errorf("LDAP connection not initialized")
	}

	// Ensure connection is alive
	if server.Connection.IsClosing() {
		if err := ReconnectToLDAPServer(server); err != nil {
			return err
		}
	}

	// Bind as the user (required for FreeIPA self-password change)
	err := server.Connection.Bind(userDN, oldPassword)
	if err != nil {
		logging.Errorf("Failed to bind as user: %s", err.Error())
		return err
	}

	// Perform password modify extended operation
	_, err = server.Connection.PasswordModify(&ldap.PasswordModifyRequest{
		UserIdentity: userDN,
		OldPassword:  oldPassword,
		NewPassword:  newPassword,
	})
	if err != nil {
		logging.Errorf("Password modify failed: %s", err.Error())
		return err
	}
	logging.Infof("Password successfully changed for %s", userDN)
	return nil
}

func CloseLDAPServer(server *LDAPServer) {
	if server != nil && server.Connection != nil {
		logging.Debug("Closing connection to LDAP server")
		err := server.Connection.Close()
		if err != nil {
			logging.Errorf("Failed to close LDAP server %s", err.Error())
		}
	}
}

func LDAPEscapeFilter(input string) string { return ldap.EscapeFilter(input) }
