package auth

import (
	"fmt"
	"html"

	mailer "github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/mail"
)

// codeEmail puts the code first in the subject so it is readable straight
// from the notification, and lets iOS offer it as one-time-code autofill.
func codeEmail(to, code string) mailer.Message {
	minutes := int(CodeTTL.Minutes())
	return mailer.Message{
		To:      to,
		Subject: fmt.Sprintf("%s is your Friend Calendar code", code),
		Text: fmt.Sprintf(
			"Your sign-in code is %s\n\nIt expires in %d minutes. If you did not try to sign in, you can ignore this email.\n",
			code, minutes,
		),
		HTML: fmt.Sprintf(`<!doctype html>
<html>
<body style="margin:0;padding:32px 16px;background:#FAF9F7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;color:#1A1918">
  <div style="max-width:420px;margin:0 auto;background:#FFFFFF;border:1px solid rgba(26,25,24,0.12);border-radius:16px;padding:32px">
    <p style="margin:0 0 8px;font-size:15px;color:#6B6862">Your sign-in code</p>
    <p style="margin:0 0 24px;font-size:34px;font-weight:700;letter-spacing:6px;color:#2E78A6">%s</p>
    <p style="margin:0;font-size:15px;line-height:20px;color:#6B6862">It expires in %d minutes. If you did not try to sign in, you can ignore this email.</p>
  </div>
</body>
</html>
`, html.EscapeString(code), minutes),
	}
}
