package main

import (
	"log"
	"net/smtp"
	"strconv"
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
	account := EmailAccount{
		email:    accountData.email,
		smtpHost: smtpHost,
		smtpPort: strconv.Itoa(smtpPort),
	}
	account.auth = smtp.PlainAuth("", accountData.username, accountData.password, smtpHost)
	return account
}

func sendEmail(account EmailAccount, toEmail []string, subject string, message string) {
	ToEmailList := ""
	for i := 0; i < len(toEmail); i++ {
		ToEmailList += toEmail[i]
		if i+1 < len(toEmail) {
			ToEmailList += ", "
		}
	}

	messageData := []byte(
		"From: " + account.email + "\r\n" +
			"To: " + ToEmailList + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			message,
	)
	err := smtp.SendMail(account.smtpHost+":"+account.smtpPort, account.auth, account.email, toEmail, messageData)
	if err != nil {
		log.Print(err)
	}
}
