package mail

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"sync"
	"time"
)

type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
	client   *smtp.Client
	mu       sync.Mutex
}

func NewMailer(host, port, username, password, from string) (*Mailer, error) {
	m := &Mailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
	
	// Establish connection early to catch configuration errors
	client, err := m.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mailer: %w", err)
	}
	
	// Send a hello to verify the connection works
	if err := client.Noop(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to verify SMTP connection: %w", err)
	}
	
	// Store the client for reuse
	m.client = client
	
	return m, nil
}

// getClient creates an initial SMTP connection for testing during initialization
func (m *Mailer) getClient() (*smtp.Client, error) {
	// Create new connection
	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	// Start TLS if available
	if ok, _ := client.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: m.host}
		if err = client.StartTLS(config); err != nil {
			client.Close()
			return nil, fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate
	auth := smtp.PlainAuth("", m.username, m.password, m.host)
	if err = client.Auth(auth); err != nil {
		client.Close()
		return nil, fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return client, nil
}

// Close closes the SMTP connection
func (m *Mailer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		err := m.client.Quit()
		m.client = nil
		return err
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

	// Lock for the entire email send operation to prevent concurrent access
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we have a valid connection
	if m.client != nil {
		// Try to verify the connection is still alive
		if err := m.client.Noop(); err != nil {
			// Connection is dead, close it and create a new one
			m.client.Close()
			m.client = nil
		}
	}

	// Create new connection if needed
	if m.client == nil {
		addr := fmt.Sprintf("%s:%s", m.host, m.port)
		client, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}

		// Start TLS if available
		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: m.host}
			if err = client.StartTLS(config); err != nil {
				client.Close()
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		}

		// Authenticate
		auth := smtp.PlainAuth("", m.username, m.password, m.host)
		if err = client.Auth(auth); err != nil {
			client.Close()
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}

		m.client = client
	}

	// Send the email using the persistent connection
	if err := m.client.Mail(m.from); err != nil {
		// Connection might be bad, reset it
		m.client.Close()
		m.client = nil
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := m.client.Rcpt(to); err != nil {
		// Reset after error
		m.client.Reset()
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := m.client.Data()
	if err != nil {
		m.client.Reset()
		return fmt.Errorf("failed to open data writer: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		w.Close()
		m.client.Reset()
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		m.client.Reset()
		return fmt.Errorf("failed to close data writer: %w", err)
	}
	
	// Reset the connection state for the next email
	if err := m.client.Reset(); err != nil {
		// If reset fails, close the connection so it gets recreated next time
		m.client.Close()
		m.client = nil
		return fmt.Errorf("failed to reset SMTP connection: %w", err)
	}

	return nil
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
		Name: "amministratore",
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
		Name: "amministratore",
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
