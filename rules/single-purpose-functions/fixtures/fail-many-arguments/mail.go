package mail

import (
	"fmt"
	"net/smtp"
)

func Send(host string, port int, from string, to string, subject string, body string) error {
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body)
	return smtp.SendMail(fmt.Sprintf("%s:%d", host, port), nil, from, []string{to}, []byte(message))
}
