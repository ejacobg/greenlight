package mailer

import (
	"bytes"
	"embed"
	"github.com/go-mail/mail/v2"
	"html/template"
	"time"
)

// Store the contents of the "templates" directory into the templateFS variable.

//go:embed "templates"
var templateFS embed.FS

type Mailer struct {
	// Used to connect to an SMTP server.
	dialer *mail.Dialer

	// Holds the name and address of the sender (e.g. "Alice Smith <alice@example.com>").
	sender string
}

func New(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second // Read/write operations should take at most 5 seconds.

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

// Send will render a template and email it to the given recipient.
func (m Mailer) Send(recipient, templateFile string, data interface{}) error {
	// Parse the templates from our embedded filesystem.
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// Render all of our templates.
	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}
	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}
	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// Create a new message with the following headers and plaintext body.
	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String()) // Always call AddAlternative() AFTER SetBody().

	err = m.dialer.DialAndSend(msg)
	if err != nil {
		return err
	}
	return nil
}
