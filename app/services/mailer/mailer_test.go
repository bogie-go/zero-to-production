package mailer_test

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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/services/mailer"
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

func startFakeSMTP(rejectTo string) *fakeSMTP {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() { _ = ln.Close() })
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

func newTestMailer(addr string) *mailer.Mailer {
	m := mailer.New(mailer.Config{Addr: addr, From: "Shop <shop@example.com>"})
	mailer.SetClock(m, func() time.Time { return time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC) })
	return m
}

func parse(raw string) *mail.Message {
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	Expect(err).NotTo(HaveOccurred(), "message does not parse:\n%s", raw)
	return msg
}

var _ = Describe("Mailer.Send", func() {
	var (
		srv *fakeSMTP
		msg mailer.Message
	)

	Context("when the server accepts the message", func() {
		BeforeEach(func() {
			srv = startFakeSMTP("")
		})

		Context("with a text message", func() {
			BeforeEach(func() {
				msg = mailer.Message{
					To: "Ann <ann@example.com>", Subject: "Order #7 — thanks", Text: "Hello Ann,\nYour order is placed. ฿100\n",
				}
			})

			It("delivers a text message", func() {
				err := newTestMailer(srv.addr).Send(context.Background(), msg)
				Expect(err).NotTo(HaveOccurred())
				Eventually(srv.done).Should(BeClosed())

				Expect(srv.from).To(Equal("shop@example.com"), "bare envelope sender")
				Expect(srv.to).To(Equal("ann@example.com"), "bare envelope recipient")

				parsed := parse(srv.data)
				subject, _ := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
				Expect(subject).To(Equal("Order #7 — thanks"))
				Expect(parsed.Header.Get("To")).To(BeElementOf(`"Ann" <ann@example.com>`, "Ann <ann@example.com>"))
				Expect(parsed.Header.Get("Message-ID")).NotTo(BeEmpty())
				Expect(parsed.Header.Get("Date")).NotTo(BeEmpty())

				body, _ := io.ReadAll(quotedprintable.NewReader(parsed.Body))
				Expect(string(body)).To(ContainSubstring("฿100"), "the UTF-8 text is intact")
			})
		})

		Context("with text and HTML", func() {
			BeforeEach(func() {
				msg = mailer.Message{
					To: "ann@example.com", Subject: "Hi", Text: "plain version", HTML: "<p>html version</p>",
				}
			})

			It("delivers an HTML alternative after the plain text", func() {
				err := newTestMailer(srv.addr).Send(context.Background(), msg)
				Expect(err).NotTo(HaveOccurred())
				Eventually(srv.done).Should(BeClosed())

				parsed := parse(srv.data)
				typ, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
				Expect(err).NotTo(HaveOccurred())
				Expect(typ).To(Equal("multipart/alternative"))

				mr := multipart.NewReader(parsed.Body, params["boundary"])
				var kinds []string
				for {
					p, err := mr.NextPart()
					if err != nil {
						break
					}
					kinds = append(kinds, strings.SplitN(p.Header.Get("Content-Type"), ";", 2)[0])
				}
				Expect(kinds).To(Equal([]string{"text/plain", "text/html"}))
			})
		})

		// Nothing in a Message may add a header or a recipient of its own.
		DescribeTable("refuses injection and bad input",
			func(bad mailer.Message) {
				Expect(newTestMailer(srv.addr).Send(context.Background(), bad)).NotTo(Succeed())
				Expect(srv.data).To(BeEmpty(), "the server received nothing")
			},
			Entry("subject with a line break", mailer.Message{To: "a@example.com", Subject: "Hi\r\nBcc: evil@example.com", Text: "x"}),
			Entry("recipient with a header", mailer.Message{To: "a@example.com\r\nBcc: evil@example.com", Subject: "Hi", Text: "x"}),
			Entry("not an address", mailer.Message{To: "not an address", Subject: "Hi", Text: "x"}),
			Entry("no text", mailer.Message{To: "a@example.com", Subject: "Hi"}),
		)
	})

	Context("when the server rejects the recipient", func() {
		BeforeEach(func() {
			srv = startFakeSMTP("gone@example.com")
			msg = mailer.Message{To: "gone@example.com", Subject: "Hi", Text: "x"}
		})

		It("surfaces the server's 550", func() {
			err := newTestMailer(srv.addr).Send(context.Background(), msg)
			Expect(err).To(MatchError(ContainSubstring("550")))
		})
	})

	Context("when the server is unreachable", func() {
		var addr string

		BeforeEach(func() {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			Expect(err).NotTo(HaveOccurred())
			addr = ln.Addr().String()
			Expect(ln.Close()).To(Succeed()) // nothing listens here now
			msg = mailer.Message{To: "a@example.com", Subject: "Hi", Text: "x"}
		})

		It("reports a connect error", func() {
			err := newTestMailer(addr).Send(context.Background(), msg)
			Expect(err).To(MatchError(ContainSubstring("connect")))
		})
	})
})
