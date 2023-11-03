package core
 
import (
    "crypto/tls"
    "gopkg.in/gomail.v2"
    "log"
)
 
type EmailConfig struct {
    SMTPHost     string
    SMTPPort     int
    SMTPUsername string
    SMTPPassword string
    InsecureSkipVerify bool
    ServerName string
}
 
// var DefaultEmailConfig = EmailConfig{
//     SMTPHost:     "smtp.ionos.co.uk",
//     SMTPPort:     587,
//     SMTPUsername: "info@podium.care",
//     SMTPPassword: "probably+All+Junk_#1",
//     InsecureSkipVerify: true,
//     ServerName: "smtp.ionos.co.uk",
// }
 
 
 
func SendMail1(from, to string, subject, body string,attachments []string, config EmailConfig) error {
    m := gomail.NewMessage()
    m.SetHeader("From", from)
    m.SetHeader("To", to)
    m.SetHeader("Subject", subject)
    m.SetBody("text/html", body)
 
    for _, attachmentPath := range attachments {
        if attachmentPath != "" {
            m.Attach(attachmentPath)
        }
    }
    
 
    d := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)
    d.TLSConfig = &tls.Config{
        InsecureSkipVerify: config.InsecureSkipVerify,
        ServerName: config.ServerName,
    }
 
    err := d.DialAndSend(m)
    if err != nil {
        log.Print(err)
    }
    return err
}