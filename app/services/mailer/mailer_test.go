package mailer

import (
	"bufio"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"
)

// fakeSMTP is just enough of an SMTP server to accept one message and keep
// what it was sent: no STARTTLS, no auth, like Mailpit in development.
type fakeSMTP struct {
	addr     string
	rejectTo string // if set, RCPT TO this address gets a 550

	from, to string
	data     string
	done     chan struct{}
}

func startFakeSMTP(t *testing.T, rejectTo string) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	f := &fakeSMTP{addr: ln.Addr().String(), rejectTo: rejectTo, done: make(chan struct{})}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer close(f.done)
		defer func() { _ = conn.Close() }()
		r := bufio.NewReader(conn)
		say := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		say("220 fake ESMTP")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				say("250 fake")
			case strings.HasPrefix(cmd, "MAIL FROM:"):
				f.from = strings.Trim(strings.TrimSpace(line[len("MAIL FROM:"):]), "<>")
				say("250 ok")
			case strings.HasPrefix(cmd, "RCPT TO:"):
				f.to = strings.Trim(strings.TrimSpace(line[len("RCPT TO:"):]), "<>")
				if f.rejectTo != "" && f.to == f.rejectTo {
					say("550 no such user")
					continue
				}
				say("250 ok")
			case cmd == "DATA":
				say("354 go ahead")
				var b strings.Builder
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if l == ".\r\n" {
						break
					}
					b.WriteString(l)
				}
				f.data = b.String()
				say("250 queued")
			case cmd == "QUIT":
				say("221 bye")
				return
			default:
				say("250 ok")
			}
		}
	}()
	return f
}

func newTestMailer(addr string) *Mailer {
	m := New(Config{Addr: addr, From: "Shop <shop@example.com>"})
	m.now = func() time.Time { return time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC) }
	return m
}

func parse(t *testing.T, raw string) *mail.Message {
	t.Helper()
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("message does not parse: %v\n%s", err, raw)
	}
	return msg
}

func TestSendDeliversATextMessage(t *testing.T) {
	srv := startFakeSMTP(t, "")
	err := newTestMailer(srv.addr).Send(context.Background(), Message{
		To: "Ann <ann@example.com>", Subject: "Order #7 — thanks", Text: "Hello Ann,\nYour order is placed. ฿100\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	<-srv.done

	if srv.from != "shop@example.com" || srv.to != "ann@example.com" {
		t.Errorf("envelope = %q -> %q, want bare addresses", srv.from, srv.to)
	}
	msg := parse(t, srv.data)
	if got, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject")); got != "Order #7 — thanks" {
		t.Errorf("Subject = %q", got)
	}
	if msg.Header.Get("To") != `"Ann" <ann@example.com>` && msg.Header.Get("To") != "Ann <ann@example.com>" {
		t.Errorf("To = %q", msg.Header.Get("To"))
	}
	if msg.Header.Get("Message-ID") == "" || msg.Header.Get("Date") == "" {
		t.Error("missing Message-ID or Date")
	}
	body, _ := io.ReadAll(quotedprintable.NewReader(msg.Body))
	if !strings.Contains(string(body), "฿100") {
		t.Errorf("body = %q, want the UTF-8 text intact", body)
	}
}

func TestSendDeliversAnHTMLAlternative(t *testing.T) {
	srv := startFakeSMTP(t, "")
	err := newTestMailer(srv.addr).Send(context.Background(), Message{
		To: "ann@example.com", Subject: "Hi", Text: "plain version", HTML: "<p>html version</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	<-srv.done

	msg := parse(t, srv.data)
	typ, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || typ != "multipart/alternative" {
		t.Fatalf("Content-Type = %q, %v", msg.Header.Get("Content-Type"), err)
	}
	mr := multipart.NewReader(msg.Body, params["boundary"])
	var kinds []string
	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}
		kinds = append(kinds, strings.SplitN(p.Header.Get("Content-Type"), ";", 2)[0])
	}
	if strings.Join(kinds, ",") != "text/plain,text/html" {
		t.Errorf("parts = %v, want plain first, html last", kinds)
	}
}

// Nothing in a Message may add a header or a recipient of its own.
func TestSendRefusesInjectionAndBadInput(t *testing.T) {
	srv := startFakeSMTP(t, "")
	m := newTestMailer(srv.addr)
	cases := map[string]Message{
		"subject with a line break": {To: "a@example.com", Subject: "Hi\r\nBcc: evil@example.com", Text: "x"},
		"recipient with a header":   {To: "a@example.com\r\nBcc: evil@example.com", Subject: "Hi", Text: "x"},
		"not an address":            {To: "not an address", Subject: "Hi", Text: "x"},
		"no text":                   {To: "a@example.com", Subject: "Hi"},
	}
	for name, msg := range cases {
		if err := m.Send(context.Background(), msg); err == nil {
			t.Errorf("%s: Send succeeded, want an error", name)
		}
	}
	if srv.data != "" {
		t.Errorf("server received %q, want nothing", srv.data)
	}
}

func TestSendReportsARejectedRecipient(t *testing.T) {
	srv := startFakeSMTP(t, "gone@example.com")
	err := newTestMailer(srv.addr).Send(context.Background(), Message{To: "gone@example.com", Subject: "Hi", Text: "x"})
	if err == nil || !strings.Contains(err.Error(), "550") {
		t.Errorf("err = %v, want the server's 550 surfaced", err)
	}
}

func TestSendReportsAnUnreachableServer(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	_ = ln.Close() // nothing listens here now

	err := newTestMailer(addr).Send(context.Background(), Message{To: "a@example.com", Subject: "Hi", Text: "x"})
	if err == nil || !strings.Contains(err.Error(), "connect") {
		t.Errorf("err = %v, want a connect error", err)
	}
}
