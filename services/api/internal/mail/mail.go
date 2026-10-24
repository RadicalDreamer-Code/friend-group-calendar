// Package mail sends transactional emails over SMTP.
//
// Locally the SMTP server is Mailpit; in production any provider with an
// SMTP endpoint works (Resend, Postmark, ...), so no vendor SDK is needed.
package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// sendTimeout bounds one delivery to the SMTP server, from dialing to QUIT.
const sendTimeout = 30 * time.Second

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
// Send upgrades to STARTTLS whenever the server offers it, and net/smtp
// refuses to send PLAIN credentials over an unencrypted connection to a
// non-local host.
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
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	if err := s.deliver(ctx, msg.To, body); err != nil {
		// The connection's deadline is the context's, but its timer can fire
		// first; wait for the context to end too, so the error says why.
		if errors.Is(err, os.ErrDeadlineExceeded) {
			<-ctx.Done()
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = ctxErr
		}
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}

// deliver is smtp.SendMail with a deadline. net/smtp has no context support,
// so the connection carries the context's deadline and is closed when the
// context ends; a server that stops answering cannot hold the request or
// leak the connection.
func (s *SMTP) deliver(ctx context.Context, to string, body []byte) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			conn.Close()
			return err
		}
	}
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
			return err
		}
	}
	if s.auth != nil {
		if ok, _ := c.Extension("AUTH"); !ok {
			return errors.New("smtp: server doesn't support AUTH")
		}
		if err := c.Auth(s.auth); err != nil {
			return err
		}
	}
	if err := c.Mail(s.from.Address); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
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
