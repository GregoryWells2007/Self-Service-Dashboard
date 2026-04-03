package ldap

import (
	"crypto/tls"
	"strings"

	"astraltech.xyz/accountmanager/src/logging"
	"github.com/go-ldap/ldap/v3"
)

type LDAPServer struct {
	URL                string
	StartTLS           bool
	IgnoreInsecureCert bool
}

func (s *LDAPServer) TestConnection() (bool, error) {
	l, err := ldap.DialURL(s.URL)
	l.Close()
	if err != nil {
		return false, err
	}
	return true, nil
}

// internal connect, should not be used regularly
func (s *LDAPServer) connect() (*ldap.Conn, error) {
	l, err := ldap.DialURL(s.URL)
	if err != nil {
		return nil, err
	}

	if s.StartTLS {
		err = l.StartTLS(&tls.Config{
			InsecureSkipVerify: s.IgnoreInsecureCert,
		})
		if err != nil {
			l.Close()
			return nil, err
		}
	}

	return l, nil
}
func (s *LDAPServer) connectAsUser(userDN, password string) (*ldap.Conn, error) {
	conn, err := s.connect()
	if err != nil {
		return nil, err
	}
	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *LDAPServer) AuthenticateUser(userDN, password string) (bool, error) {
	conn, err := s.connectAsUser(userDN, password)
	if err != nil || conn == nil {
		return false, err
	}
	conn.Close()
	return true, nil
}

func (s *LDAPServer) SerchServer(
	userDN string, password string,
	baseDN string,
	searchFilter string, attributes []string,
) (*LDAPSearch, error) {
	logging.Debugf("Searching %s LDAP server\n\tBase DN: %s\n\tSearch Filter %s\n\tAttributes: %s", s.URL, baseDN, searchFilter, strings.Join(attributes, ","))
	conn, err := s.connectAsUser(userDN, password)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		searchFilter, attributes,
		nil,
	)
	sr, err := conn.Search(searchRequest)
	if err != nil {
		logging.Errorf("Failed to search LDAP server %s\n", err.Error())
		return nil, err
	}
	return &LDAPSearch{sr}, nil
}

func (s *LDAPServer) ChangePassword(userDN string, oldPassword string, newPassword string) error {
	conn, err := s.connectAsUser(userDN, oldPassword)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Perform password modify extended operation
	_, err = conn.PasswordModify(&ldap.PasswordModifyRequest{
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

func (s *LDAPServer) ModifyAttribute(bindUserDN string, password string, userDN string, attribute string, data []string) error {
	logging.Infof("Modifing LDAP attribute %s", attribute)

	conn, err := s.connectAsUser(bindUserDN, password)
	if err != nil {
		return err
	}
	defer conn.Close()

	modify := ldap.NewModifyRequest(userDN, nil)
	modify.Replace(attribute, data)
	err = conn.Modify(modify)
	if err != nil {
		logging.Errorf("Failed to modify %s", err.Error())
		return err
	}
	return nil
}
