// Package mailer sends email over plain SMTP, using only the standard
// library. In development the server is Mailpit from docker-compose, which
// catches every message for you to read at http://localhost:8125; in
// production it is your provider's SMTP endpoint. The code is identical:
// only config (smtp.address, smtp.username, smtp.password, mail.from) changes.
//
// A *Mailer is safe for concurrent use. Send blocks until the server has
// accepted the message or the context ends, so call it from a job, not from
// a request (a slow or failing mail server must never slow or fail a login).
package mailer

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// Config is where mail is delivered and who it is from.
type Config struct {
	Addr     string // host:port
	User     string // optional: a provider's login
	Password string
	From     string // "Name <addr>" or a bare address
}

// Message is one email. Text is required; HTML is optional, and when set the
// message goes out as multipart/alternative with Text as the fallback.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Mailer delivers Messages.
type Mailer struct {
	cfg Config
	// now and the dial timeout are fields so tests need not wait or sleep.
	now func() time.Time
}

// New builds a Mailer. It does not connect: the first Send does.
func New(cfg Config) *Mailer { return &Mailer{cfg: cfg, now: time.Now} }

// sendTimeout bounds a whole delivery when the caller's context has no
// deadline of its own.
const sendTimeout = 30 * time.Second

// Send delivers msg. An error means the server did not accept it: nothing
// was sent, and it is safe to retry.
func (m *Mailer) Send(ctx context.Context, msg Message) error {
	body, from, to, err := m.build(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	host, _, err := net.SplitHostPort(m.cfg.Addr)
	if err != nil {
		return fmt.Errorf("mailer: smtp address %q: %w", m.cfg.Addr, err)
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", m.cfg.Addr)
	if err != nil {
		return fmt.Errorf("mailer: connect to %s: %w", m.cfg.Addr, err)
	}
	// net/smtp has no context, so the deadline rides on the connection.
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mailer: smtp greeting from %s: %w", m.cfg.Addr, err)
	}
	defer func() { _ = c.Close() }()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("mailer: starttls: %w", err)
		}
	}
	if m.cfg.User != "" {
		// PlainAuth refuses to send a password over an unencrypted
		// connection to anything but localhost, which is what we want.
		if err := c.Auth(smtp.PlainAuth("", m.cfg.User, m.cfg.Password, host)); err != nil {
			return fmt.Errorf("mailer: auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("mailer: MAIL FROM %s: %w", from, err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("mailer: RCPT TO %s: %w", to, err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("mailer: DATA: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("mailer: write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: message not accepted: %w", err)
	}
	_ = c.Quit()
	return nil
}

// build renders msg to RFC 5322 bytes and returns the bare envelope
// addresses. Addresses are parsed, never concatenated, and a subject with a
// line break is refused outright: either would let a name or a subject taken
// from user input add headers of its own.
func (m *Mailer) build(msg Message) (body []byte, from, to string, err error) {
	if strings.TrimSpace(msg.Text) == "" {
		return nil, "", "", errors.New("mailer: message has no text")
	}
	if strings.ContainsAny(msg.Subject, "\r\n") {
		return nil, "", "", errors.New("mailer: subject contains a line break")
	}
	f, err := mail.ParseAddress(m.cfg.From)
	if err != nil {
		return nil, "", "", fmt.Errorf("mailer: from address %q: %w", m.cfg.From, err)
	}
	t, err := mail.ParseAddress(msg.To)
	if err != nil {
		return nil, "", "", fmt.Errorf("mailer: to address %q: %w", msg.To, err)
	}

	var b strings.Builder
	h := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	h("From", f.String())
	h("To", t.String())
	h("Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	h("Date", m.now().Format(time.RFC1123Z))
	h("Message-ID", "<"+randomID()+"@"+domainOf(f.Address)+">")
	h("MIME-Version", "1.0")

	if msg.HTML == "" {
		h("Content-Type", `text/plain; charset="utf-8"`)
		h("Content-Transfer-Encoding", "quoted-printable")
		b.WriteString("\r\n" + qp(msg.Text))
		return []byte(b.String()), f.Address, t.Address, nil
	}

	mw := multipart.NewWriter(&b)
	h("Content-Type", `multipart/alternative; boundary="`+mw.Boundary()+`"`)
	b.WriteString("\r\n")
	for _, part := range []struct{ typ, content string }{
		{"text/plain", msg.Text}, // plain first: the last part is the preferred one
		{"text/html", msg.HTML},
	} {
		pw, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {part.typ + `; charset="utf-8"`},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return nil, "", "", fmt.Errorf("mailer: build message: %w", err)
		}
		_, _ = pw.Write([]byte(qp(part.content)))
	}
	if err := mw.Close(); err != nil {
		return nil, "", "", fmt.Errorf("mailer: build message: %w", err)
	}
	return []byte(b.String()), f.Address, t.Address, nil
}

// qp encodes a body as quoted-printable, which any SMTP server accepts
// whatever 8-bit extensions it offers, and which also normalizes line
// endings to CRLF.
func qp(s string) string {
	var b strings.Builder
	w := quotedprintable.NewWriter(&b)
	_, _ = w.Write([]byte(strings.ReplaceAll(s, "\r\n", "\n")))
	_ = w.Close()
	return b.String()
}

func randomID() string {
	var raw [12]byte
	_, _ = rand.Read(raw[:])
	return hex.EncodeToString(raw[:])
}

func domainOf(addr string) string {
	if i := strings.LastIndex(addr, "@"); i >= 0 {
		return addr[i+1:]
	}
	return "localhost"
}
