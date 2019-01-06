package smtputils

import (
	"github.com/exlskills/eocsutil/config"
	"github.com/scorredoira/email"
	"net/mail"
	"net/smtp"
)

var Log = config.Cfg().GetLogger()

func SendEmail(toStr string, subject string, htmlBody string) error {
	Log.Debug("In SendEmail")
	auth := smtp.PlainAuth("", config.Cfg().SMTPUserName, config.Cfg().SMTPPassword, config.Cfg().SMTPHost)
	m := email.NewHTMLMessage(subject, htmlBody)
	m.From = mail.Address{Name: config.Cfg().SMTPFromName, Address: config.Cfg().SMTPFromAddress}
	m.To = []string{toStr}
	Log.Info("Sending Email to ",toStr)
	err := email.Send(config.Cfg().SMTPConnectionString, auth, m)
	if err != nil {
		Log.Errorf("Error sending email to %s. %v",toStr,err)
	}
	return err
}
