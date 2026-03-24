package main

import (
	"encoding/json"
	"os"
)

type LDAPConfig struct {
	LDAPURL           string `json:"ldap_url"`
	BaseDN            string `json:"base_dn"`
	BindDN            string `json:"bind_dn"`
	BindPassword      string `json:"bind_password"`
	Security          string `json:"security"`
	IgnoreInvalidCert bool   `json:"ignore_invalid_cert"`
}

type StyleConfig struct {
	FaviconPath string `json:"favicon_path"`
	LogoPath    string `json:"logo_path"`
}

type WebserverConfig struct {
	Port int `json:"port"`
}

type ServerConfig struct {
	LDAPConfig      LDAPConfig      `json:"ldap_config"`
	StyleConfig     StyleConfig     `json:"style_config"`
	WebserverConfig WebserverConfig `json:"server_config"`
}

func loadServerConfig(path string) (*ServerConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ServerConfig
	err = json.Unmarshal(file, &cfg)
	return &cfg, err
}
