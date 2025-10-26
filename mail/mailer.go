package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"strings"
	"time"
)

var itMonths = map[string]string{
	"January":   "Gennaio",
	"February":  "Febbraio",
	"March":     "Marzo",
	"April":     "Aprile",
	"May":       "Maggio",
	"June":      "Giugno",
	"July":      "Luglio",
	"August":    "Agosto",
	"September": "Settembre",
	"October":   "Ottobre",
	"November":  "Novembre",
	"December":  "Dicembre",
}

type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
	mailCh   chan *mailMessage
	client   *smtp.Client
}

type mailMessage struct {
	to      string
	subject string
	data    EmailData
	respCh  chan error
}

func NewMailer(host, port, username, password, from string) (*Mailer, error) {
	client, err := connect(host, port, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mailer: %w", err)
	}

	m := &Mailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		mailCh:   make(chan *mailMessage, 100), // Buffered channel for better performance
		client:   client,
	}

	return m, nil
}

// Run starts the mail sending loop. Should be called in a goroutine.
func (m *Mailer) Run(ctx context.Context) {
	ticker := time.Tick(time.Second * 10)
	defer close(m.mailCh)
	for {
		select {
		case <-ctx.Done():
			// Clean shutdown
			if m.client != nil {
				m.client.Quit()
			}
			return

		case <-ticker:
			if m.client != nil {
				if err := m.client.Noop(); err == nil {
					continue
				}

				m.client.Close()
				m.client = nil
			}

			client, err := connect(m.host, m.port, m.username, m.password)
			if err != nil {
				log.Printf("cannot connect to SMTP server: %v", err)
			}

			m.client = client

		case msg := <-m.mailCh:
			// Send the email
			err := m.sendEmail(m.client, msg.to, msg.subject, msg.data)
			msg.respCh <- err

			// If there was an error, close connection so it reconnects next time
			if err != nil {
				log.Printf("smtp client error: %v", err)
			}
		}
	}
}

// connect creates a new SMTP connection
func connect(host, port, username, password string) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}

	// Start TLS if available
	if ok, _ := client.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: host}
		if err = client.StartTLS(config); err != nil {
			client.Close()
			return nil, err
		}
	}

	// Authenticate
	auth := smtp.PlainAuth("", username, password, host)
	if err = client.Auth(auth); err != nil {
		client.Close()
		return nil, err
	}

	// Send a hello to verify the connection works
	if err := client.Noop(); err != nil {
		return nil, fmt.Errorf("failed to verify SMTP connection: %w", err)
	}

	return client, nil
}

// sendEmail sends an email using the provided SMTP client
func (m *Mailer) sendEmail(client *smtp.Client, to, subject string, data EmailData) error {
	if data.AppName == "" {
		data.AppName = "Wellness & Nutrition"
	}
	if data.AppLink == "" {
		data.AppLink = os.Getenv("AUTH_URL")
	}
	if data.Copyright == "" {
		data.Copyright = "Tutti i diritti riservati"
	}

	tmpl, err := template.New("email").Parse(emailTemplate)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", m.from, to, subject, body.String())

	// Send the email using the SMTP client
	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		client.Reset()
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		client.Reset()
		return fmt.Errorf("failed to open data writer: %w", err)
	}
	defer w.Close()

	if _, err := w.Write([]byte(msg)); err != nil {
		w.Close()
		client.Reset()
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

type EmailData struct {
	Name         string
	Intro        string
	ButtonText   string
	ButtonLink   string
	Instructions string
	Signature    string
	Outro        string
	Title        string
	AppName      string
	AppLink      string
	Copyright    string
}

const emailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            text-align: center;
            padding: 20px 0;
            border-bottom: 2px solid #22BC66;
        }
        .content {
            padding: 30px 0;
        }
        .button {
            display: inline-block;
            padding: 12px 24px;
            background-color: #22BC66;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            margin: 20px 0;
        }
        .footer {
            text-align: center;
            padding: 20px 0;
            border-top: 1px solid #ddd;
            color: #666;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.AppName}}</h1>
    </div>
    <div class="content">
        {{if .Name}}<p>Ciao {{.Name}},</p>{{end}}
        {{if .Intro}}<p>{{.Intro}}</p>{{end}}
        {{if .Title}}<h2>{{.Title}}</h2>{{end}}
        {{if .Instructions}}<p>{{.Instructions}}</p>{{end}}
        {{if .ButtonText}}
        <p style="text-align: center;">
            <a href="{{.ButtonLink}}" class="button">{{.ButtonText}}</a>
        </p>
        {{end}}
        {{if .Outro}}<p>{{.Outro}}</p>{{end}}
        {{if .Signature}}<p>{{.Signature}}</p>{{end}}
    </div>
    <div class="footer">
        <p>{{.Copyright}}</p>
        <p><a href="{{.AppLink}}">{{.AppName}}</a></p>
    </div>
</body>
</html>
`

func (m *Mailer) SendEmail(to, subject string, data EmailData) error {
	respCh := make(chan error, 1)
	msg := &mailMessage{
		to:      to,
		subject: subject,
		data:    data,
		respCh:  respCh,
	}

	m.mailCh <- msg
	return <-respCh
}

func (m *Mailer) SendWelcomeEmail(email, firstName, verificationURL string) error {
	data := EmailData{
		Name:         firstName,
		Intro:        "Benvenuto in Wellness & Nutrition.",
		Instructions: "Per verificare il tuo account e impostare una password, clicca il pulsante di seguito:",
		ButtonText:   "Conferma account",
		ButtonLink:   verificationURL,
		Signature:    "Grazie per averci scelto",
		Outro:        fmt.Sprintf("Hai bisogno di aiuto? Invia un messaggio a %s e saremo felici di aiutarti", os.Getenv("EMAIL_NOTIFY_ADDRESS")),
	}

	return m.SendEmail(email, "Benvenuto in Wellness & Nutrition", data)
}

func (m *Mailer) SendResetEmail(email, firstName, verificationURL string) error {
	data := EmailData{
		Name:         firstName,
		Intro:        "Ricevi questa email per ripristinare la credenziali.",
		Instructions: "Per ripristinare le credenziali, clicca il pulsante di seguito:",
		ButtonText:   "Ripristina credenziali",
		ButtonLink:   verificationURL,
		Signature:    "Grazie per averci scelto",
		Outro:        fmt.Sprintf("Hai bisogno di aiuto? Invia un messaggio a %s e saremo felici di aiutarti", os.Getenv("EMAIL_NOTIFY_ADDRESS")),
	}

	return m.SendEmail(email, "Ripristino password", data)
}

func (m *Mailer) SendNewBookingNotification(firstName, lastName string, startsAt time.Time) error {
	notifyEmail := os.Getenv("EMAIL_NOTIFY_ADDRESS")
	localTime, err := formatUserTime(startsAt, "Europe/Rome")
	if err != nil {
		return err
	}

	data := EmailData{
		Name: "amministratore",
		Intro: fmt.Sprintf("Una nuova prenotazione è stata inserita da %s %s per %s",
			firstName, lastName, localTime),
		Title:     "Nuova prenotazione",
		Signature: "Saluti,",
	}

	return m.SendEmail(notifyEmail, "Nuova prenotazione", data)
}

func (m *Mailer) SendDeleteBookingNotification(firstName, lastName string, startsAt time.Time) error {
	notifyEmail := os.Getenv("EMAIL_NOTIFY_ADDRESS")

	localTime, err := formatUserTime(startsAt, "Europe/Rome")
	if err != nil {
		return err
	}

	data := EmailData{
		Name: "amministratore",
		Intro: fmt.Sprintf("Una prenotazione è stata cancellata da %s %s per %s",
			firstName, lastName, localTime),
		Title:     "Prenotazione cancellata",
		Signature: "Saluti,",
	}

	return m.SendEmail(notifyEmail, "Prenotazione cancellata", data)
}

func (m *Mailer) SendReminderEmail(email, firstName string, startsAt time.Time) error {
	localTime, err := formatUserTime(startsAt, "Europe/Rome")
	if err != nil {
		return err
	}

	data := EmailData{
		Name:         firstName,
		Intro:        "Questo è un promemoria per la tua prossima prenotazione.",
		Title:        "Promemoria prenotazione",
		Instructions: fmt.Sprintf("La tua prenotazione è prevista per %s", localTime),
		Outro:        "Ti aspettiamo!",
		Signature:    "Grazie per averci scelto",
	}

	return m.SendEmail(email, "Promemoria prenotazione - Wellness & Nutrition", data)
}

func formatUserTime(t time.Time, tz string) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", err
	}
	lt := t.In(loc)
	english := lt.Format("02 January 2006 alle 15:04")
	return strings.ReplaceAll(english, lt.Month().String(), itMonths[lt.Month().String()]), nil
}
