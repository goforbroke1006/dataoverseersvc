package mailing

import (
	"net/smtp"
	"fmt"
)

type Mailer struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewGmailMailer(username, password string) *Mailer {
	return &Mailer{
		host:     "smtp.gmail.com",
		port:     587,
		username: username,
		password: password,
		from:     username,
	}
}

func NewMailer(host string, port int, username, password, from string) *Mailer {
	return &Mailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (m Mailer) Send(to, title, body string) error {
	msg := "From: " + m.from + "\n" + "To: " + to + "\n" + "Subject: " + title + "\n\n" + body

	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", m.host, m.port),
		smtp.PlainAuth("", m.username, m.password, m.host),
		m.from,
		[]string{to},
		[]byte(msg),
	)

	if err != nil {
		return err
	}

	return nil
}
