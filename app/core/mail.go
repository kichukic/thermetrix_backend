package core

import (
	"crypto/tls"
	"gopkg.in/gomail.v2"
	"log"
)

const (
	/*SMTP_HOST     = "mail.podium.care"
	SMTP_PORT     = 587
	SMTP_USERNAME = "info@podium.care"
	SMTP_PASSWORD = "w-fDhrC4e"*/
	SMTP_HOST     = "smtp.ionos.co.uk"
	SMTP_PORT     = 587
	SMTP_USERNAME = "info@podium.care"
	SMTP_PASSWORD = "probably+All+Junk_#1"
)

func SendMail(from string, to []string, cc []string, bcc []string, subject string, body string, files []string) error {

	if from == "" {
		// holen aus Company Settings
		from = "apps@symblcrowd.de"
	}

	log.Println(from)
	log.Println(to)
	log.Println(cc)
	log.Println(bcc)
	log.Println(subject)
	log.Println(body)
	log.Println(files)

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to...)
	m.SetHeader("Cc", cc...)
	m.SetHeader("Bcc", bcc...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	if files != nil {
		for _, file := range files {
			m.Attach(file)
		}
	}
	host := SMTP_HOST
	port := SMTP_PORT
	username := SMTP_USERNAME
	password := SMTP_PASSWORD

	if Config.MailServer.SmtpPort > 0 && Config.MailServer.SmtpPassword != "" && Config.MailServer.SmtpHost != "" && Config.MailServer.SmtpUsername != "" {
		host = Config.MailServer.SmtpHost
		port = Config.MailServer.SmtpPort
		username = Config.MailServer.SmtpUsername
		password = Config.MailServer.SmtpPassword
	}

	d := gomail.NewDialer(host, port, username, password)

	//d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true, ServerName: "smtp.ionos.co.uk"} //extendcp.co.uk
	// Send the email to Bob, Cora and Dan.
	err := d.DialAndSend(m)
	if err != nil {
		log.Print(err)
	}
	return err
}
