package main

import (
	"fmt"
	"time"

	"astraltech.xyz/accountmanager/src/email"
	"astraltech.xyz/accountmanager/src/logging"
	"astraltech.xyz/accountmanager/src/worker"
)

func InitPasswordExpiry() {
	go func() {
		CheckPasswordExpriy()
	}()
	worker.CreateWorker(time.Hour*12, CheckPasswordExpriy)
}

func CheckPasswordExpriy() {
	logging.Infof("Starting password expiry check")

	now := time.Now().UTC()
	formatted := now.Format("20060102150405Z")

	search, err := ldapServer.SerchServer(serverConfig.LDAPConfig.BindDN, serverConfig.LDAPConfig.BindPassword, serverConfig.LDAPConfig.BaseDN, fmt.Sprintf("(&(objectclass=person)(krbPasswordExpiration<=%s))", formatted), []string{"cn", "mail", "krbPasswordExpiration"})
	if err != nil {
		logging.Warn(err.Error())
	}

	logging.Infof("%d users with expired passwords", search.EntryCount())

	for i := range search.EntryCount() {
		emailAddr := search.GetEntry(i).GetAttributeValue("mail")
		if len(emailAddr) <= 0 {
			continue
		}

		t, err := time.Parse("20060102150405Z", search.GetEntry(i).GetAttributeValue("krbPasswordExpiration"))
		if err != nil {
			panic(err)
		}
		formatted := t.Format("January 2, 2006 at 3:04 PM MST")

		data := map[string]any{
			"Username":    search.GetEntry(i).GetAttributeValue("cn"),
			"ExpiredAt":   formatted,
			"ResetURL":    "https://example.com/reset?token=abc123",
			"ServiceName": "Astral Tech",
		}

		email_template, err := email.RenderTemplate("./data/email-templates/expired-password.html", data, nil)
		if err != nil {
			logging.Errorf("Failed to load email template: %s", err.Error())
		}
		noReplyEmail.SendEmail([]string{emailAddr}, "Password expired", email_template)
	}
}
