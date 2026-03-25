package main

import (
	"net/smtp"
	"strconv"
	"strings"

	"astraltech.xyz/accountmanager/src/logging"
)

type EmailAccount struct {
	auth     smtp.Auth
	email    string
	smtpHost string
	smtpPort string
}

type EmailAccountData struct {
	username string
	password string
	email    string
}

func createEmailAccount(accountData EmailAccountData, smtpHost string, smtpPort int) EmailAccount {
	logging.Debugf("Creating Email Account: \n\tUsername: %s\n\tEmail: %s\n\tSMTP Host: %s:%d", accountData.username, accountData.email, smtpHost, smtpPort)
	account := EmailAccount{
		email:    accountData.email,
		smtpHost: smtpHost,
		smtpPort: strconv.Itoa(smtpPort),
	}
	account.auth = smtp.PlainAuth("", accountData.username, accountData.password, smtpHost)
	return account
}

func sendEmail(account EmailAccount, toEmail []string, subject string, message string) {
	logging.Debugf("Sending an email from %s to %s", account.email, strings.Join(toEmail, ""))

	ToEmailList := strings.Join(toEmail, "")

	messageData := []byte(
		"From: " + account.email + "\r\n" +
			"To: " + ToEmailList + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			message,
	)
	err := smtp.SendMail(account.smtpHost+":"+account.smtpPort, account.auth, account.email, toEmail, messageData)
	if err != nil {
		logging.Error("Failed to send email")
		logging.Error(err.Error())
	}
}
