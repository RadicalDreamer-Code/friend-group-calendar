// Package mail sends transactional emails over SMTP.
//
// Locally the SMTP server is Mailpit; in production any provider with an
// SMTP endpoint works (Resend, Postmark, ...), so no vendor SDK is needed.
package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Sender is what the rest of the API depends on, so tests can capture mail.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

type SMTP struct {
	addr string
	host string
	auth smtp.Auth
	from *mail.Address
}

// NewSMTP prepares a sender. Credentials are optional (Mailpit needs none).
// net/smtp upgrades to STARTTLS whenever the server offers it and refuses to
// send PLAIN credentials over an unencrypted connection to a non-local host.
func NewSMTP(host string, port int, username, password, from string) (*SMTP, error) {
	fromAddr, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("parse MAIL_FROM: %w", err)
	}
	s := &SMTP{
		addr: net.JoinHostPort(host, strconv.Itoa(port)),
		host: host,
		from: fromAddr,
	}
	if username != "" {
		s.auth = smtp.PlainAuth("", username, password, host)
	}
	return s, nil
}

func (s *SMTP) Send(ctx context.Context, msg Message) error {
	body, err := s.build(msg)
	if err != nil {
		return err
	}
	// net/smtp has no context support; run it in a goroutine so a hanging
	// server cannot hold the request past its deadline.
	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(s.addr, s.auth, s.from.Address, []string{msg.To}, body) }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("send mail: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("send mail: %w", ctx.Err())
	}
}

// build renders a multipart/alternative message with a text and an HTML part.
func (s *SMTP) build(msg Message) ([]byte, error) {
	to, err := mail.ParseAddress(msg.To)
	if err != nil {
		return nil, fmt.Errorf("parse recipient: %w", err)
	}
	boundary := randomHex(12)

	var b bytes.Buffer
	header := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	header("From", s.from.String())
	header("To", to.String())
	header("Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("Message-ID", fmt.Sprintf("<%s@%s>", randomHex(16), domainOf(s.from.Address)))
	header("MIME-Version", "1.0")
	header("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, boundary))
	b.WriteString("\r\n")

	for _, part := range []struct{ contentType, content string }{
		{"text/plain", msg.Text},
		{"text/html", msg.HTML},
	} {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		fmt.Fprintf(&b, "Content-Type: %s; charset=utf-8\r\n", part.contentType)
		b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		qp := quotedprintable.NewWriter(&b)
		if _, err := qp.Write([]byte(part.content)); err != nil {
			return nil, err
		}
		if err := qp.Close(); err != nil {
			return nil, err
		}
		b.WriteString("\r\n")
	}
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.Bytes(), nil
}

func domainOf(addr string) string {
	if i := strings.LastIndexByte(addr, '@'); i >= 0 {
		return addr[i+1:]
	}
	return "localhost"
}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf) // crypto/rand.Read never returns an error
	return hex.EncodeToString(buf)
}
