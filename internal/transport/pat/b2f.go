package pat

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/4current/relayops/internal/core"
)

func BuildB2F(mycall, mid string, m *core.Message) ([]byte, error) {
	to := firstTo(m)
	if to == "" {
		return nil, fmt.Errorf("b2f: missing To")
	}
	from := strings.TrimSpace(m.From.Callsign)
	if from == "" {
		from = mycall
	}
	if strings.TrimSpace(from) == "" {
		return nil, fmt.Errorf("b2f: missing From")
	}

	body := normalizeLF(m.Body)
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	bodyBytes := []byte(body)

	date := time.Now().UTC().Format("2006/01/02 15:04")

	h := ""
	h += fmt.Sprintf("Mid: %s\n", mid)
	h += fmt.Sprintf("Body: %d\n", len(bodyBytes))
	h += "Content-Transfer-Encoding: 8bit\n"
	h += "Content-Type: text/plain; charset=ISO-8859-1\n"
	h += fmt.Sprintf("Date: %s\n", date)
	h += fmt.Sprintf("From: %s\n", strings.ToUpper(from))
	h += fmt.Sprintf("Mbo: %s\n", strings.ToUpper(mycall))
	h += fmt.Sprintf("Subject: %s\n", sanitizeHeader(m.Subject))
	h += fmt.Sprintf("To: %s\n", strings.ToUpper(to))
	h += "Type: Private\n"
	h += "\n"

	return append([]byte(h), bodyBytes...), nil
}

func NewMID(n int) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

func firstTo(m *core.Message) string {
	for _, a := range m.To {
		if s := strings.TrimSpace(a.Callsign); s != "" {
			return s
		}
		if s := strings.TrimSpace(a.Email); s != "" {
			if i := strings.IndexByte(s, '@'); i > 0 {
				return s[:i]
			}
			return s
		}
	}
	return ""
}

func normalizeLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
