package email

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
	Username string
	Password string
	Email    string
}

func CreateEmailAccount(accountData EmailAccountData, smtpHost string, smtpPort int) EmailAccount {
	logging.Debugf("Creating Email Account: \n\tUsername: %s\n\tEmail: %s\n\tSMTP Host: %s:%d", accountData.Username, accountData.Email, smtpHost, smtpPort)
	account := EmailAccount{
		email:    accountData.Email,
		smtpHost: smtpHost,
		smtpPort: strconv.Itoa(smtpPort),
	}
	account.auth = smtp.PlainAuth("", accountData.Username, accountData.Password, smtpHost)
	return account
}

func (account *EmailAccount) SendEmail(toEmails []string, subject string, message string) {
	logging.Debugf("Sending an email from %s to %s", account.email, strings.Join(toEmails, ""))

	ToEmailList := strings.Join(toEmails, "")

	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"

	messageData := []byte(
		"From: " + account.email + "\r\n" +
			"To: " + ToEmailList + "\r\n" +
			"Subject: " + subject + "\r\n" +
			mime +
			"\r\n" +
			message,
	)
	err := smtp.SendMail(account.smtpHost+":"+account.smtpPort, account.auth, account.email, toEmails, messageData)
	if err != nil {
		logging.Error("Failed to send email")
		logging.Error(err.Error())
	}
}
