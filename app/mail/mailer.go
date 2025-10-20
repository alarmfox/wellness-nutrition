package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"time"
)

type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewMailer() *Mailer {
	return &Mailer{
		host:     os.Getenv("EMAIL_SERVER_HOST"),
		port:     os.Getenv("EMAIL_SERVER_PORT"),
		username: os.Getenv("EMAIL_SERVER_USER"),
		password: os.Getenv("EMAIL_SERVER_PASSWORD"),
		from:     os.Getenv("EMAIL_FROM"),
	}
}

type EmailData struct {
	Name          string
	Intro         string
	ButtonText    string
	ButtonLink    string
	Instructions  string
	Signature     string
	Outro         string
	Title         string
	AppName       string
	AppLink       string
	Copyright     string
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
	if data.AppName == "" {
		data.AppName = "Wellness & Nutrition"
	}
	if data.AppLink == "" {
		data.AppLink = os.Getenv("NEXTAUTH_URL")
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
	
	auth := smtp.PlainAuth("", m.username, m.password, m.host)
	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	
	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
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
	
	data := EmailData{
		Name:  "amministratore",
		Intro: fmt.Sprintf("Una nuova prenotazione è stata inserita da %s %s per %s", 
			firstName, lastName, startsAt.Format("02 Jan 2006 15:04")),
		Title:     "Nuova prenotazione",
		Signature: "Saluti,",
	}
	
	return m.SendEmail(notifyEmail, "Nuova prenotazione", data)
}

func (m *Mailer) SendDeleteBookingNotification(firstName, lastName string, startsAt time.Time) error {
	notifyEmail := os.Getenv("EMAIL_NOTIFY_ADDRESS")
	
	data := EmailData{
		Name:  "amministratore",
		Intro: fmt.Sprintf("Una prenotazione è stata cancellata da %s %s per %s", 
			firstName, lastName, startsAt.Format("02 Jan 2006 15:04")),
		Title:     "Prenotazione cancellata",
		Signature: "Saluti,",
	}
	
	return m.SendEmail(notifyEmail, "Prenotazione cancellata", data)
}

func (m *Mailer) SendReminderEmail(email, firstName string, startsAt time.Time) error {
	data := EmailData{
		Name:         firstName,
		Intro:        "Questo è un promemoria per la tua prossima prenotazione.",
		Title:        "Promemoria prenotazione",
		Instructions: fmt.Sprintf("La tua prenotazione è prevista per %s", startsAt.Format("02 January 2006 alle 15:04")),
		Outro:        "Ti aspettiamo!",
		Signature:    "Grazie per averci scelto",
	}
	
	return m.SendEmail(email, "Promemoria prenotazione - Wellness & Nutrition", data)
}
