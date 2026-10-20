package mail

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func newTestSender(t *testing.T, ln net.Listener) *SMTP {
	t.Helper()
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	p, _ := strconv.Atoi(port)
	s, err := NewSMTP(host, p, "", "", "Friend Calendar <hello@example.com>")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	return ln
}

var testMessage = Message{To: "ada@example.com", Subject: "123456 is your code", Text: "Your code is 123456", HTML: "<p>123456</p>"}

// A server that accepts the connection and never answers must not hold the
// request, and the connection must be closed rather than leaked.
func TestSendGivesUpOnUnresponsiveServer(t *testing.T) {
	ln := listen(t)
	closed := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, err = conn.Read(make([]byte, 1)) // returns once the client hangs up
		closed <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := newTestSender(t, ln).Send(ctx, testMessage)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("Send took %v after its deadline", d)
	}
	select {
	case err := <-closed:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("server read: %v, want EOF from the closed connection", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the connection to the server was left open")
	}
}

func TestSendDeliversMessage(t *testing.T) {
	ln := listen(t)
	type received struct {
		from, to, data string
	}
	got := make(chan received, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		got <- serveOneMessage(conn)
	}()

	if err := newTestSender(t, ln).Send(context.Background(), testMessage); err != nil {
		t.Fatal(err)
	}
	r := <-got
	if r.from != "<hello@example.com>" || r.to != "<ada@example.com>" {
		t.Fatalf("envelope from %s to %s", r.from, r.to)
	}
	if !strings.Contains(r.data, "Subject: 123456 is your code") || !strings.Contains(r.data, "Your code is 123456") {
		t.Fatalf("message:\n%s", r.data)
	}
}

// serveOneMessage plays a minimal SMTP server for one message.
func serveOneMessage(conn net.Conn) (r struct{ from, to, data string }) {
	in := bufio.NewReader(conn)
	reply := func(s string) { io.WriteString(conn, s+"\r\n") }
	reply("220 test ESMTP")
	for {
		line, err := in.ReadString('\n')
		if err != nil {
			return r
		}
		cmd := strings.TrimRight(line, "\r\n")
		switch upper := strings.ToUpper(cmd); {
		case strings.HasPrefix(upper, "EHLO"):
			reply("250 test")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			r.from = cmd[len("MAIL FROM:"):]
			reply("250 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			r.to = cmd[len("RCPT TO:"):]
			reply("250 OK")
		case upper == "DATA":
			reply("354 go ahead")
			var data strings.Builder
			for {
				l, err := in.ReadString('\n')
				if err != nil || l == ".\r\n" {
					break
				}
				data.WriteString(l)
			}
			r.data = data.String()
			reply("250 OK")
		case upper == "QUIT":
			reply("221 bye")
			return r
		default:
			reply("502 not implemented")
		}
	}
}
