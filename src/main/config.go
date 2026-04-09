package main

import (
	"encoding/json"
	"os"

	"astraltech.xyz/accountmanager/src/logging"
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
	Port    int    `json:"port"`
	BaseURL string `json:"base_url"`
}

type ServerConfig struct {
	LDAPConfig      LDAPConfig      `json:"ldap_config"`
	StyleConfig     StyleConfig     `json:"style_config"`
	WebserverConfig WebserverConfig `json:"server_config"`
}

func loadServerConfig(path string) (*ServerConfig, error) {
	logging.Debugf("Loading server config file: %s", path)
	file, err := os.ReadFile(path)
	if err != nil {
		logging.Errorf("Failed to load server config")
		logging.Error(err.Error())
		return nil, err
	}

	var cfg ServerConfig
	logging.Debugf("Unmarshaling JSON data")
	err = json.Unmarshal(file, &cfg)
	if err != nil {
		logging.Error("Failed to read JSON data")
		logging.Error(err.Error())
	}
	return &cfg, nil
}
