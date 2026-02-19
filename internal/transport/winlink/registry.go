package winlink

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RegistryRecord is a best-effort parse of a single Winlink Express registry row.
// Winlink Express (RMS Express) persists message state in Data/Registry.txt, with fields delimited by ASCII 0x01.
type RegistryRecord struct {
	ID        string
	CreatedAt time.Time
	From      string
	To        string
	MsgNum    string
	Subject   string
	Folder    string
	State     string // e.g. "Sent" (best-effort)
	Freq      string // best-effort (often numeric kHz/Hz-ish)
	GPS       string // best-effort (trailing field in your samples)
	RawFields []string
}

// ReadRegistry reads <root>/Data/Registry.txt and returns parsed records.
func ReadRegistry(root string) ([]RegistryRecord, error) {
	path := filepath.Join(root, "Data", "Registry.txt")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open Registry.txt: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Registry lines can be long; bump the buffer.
	buf := make([]byte, 0, 1024*64)
	scanner.Buffer(buf, 1024*1024)

	var out []RegistryRecord
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\x01")
		if len(fields) < 9 {
			continue // ignore short/unknown rows
		}

		r := RegistryRecord{RawFields: fields}
		r.ID = strings.TrimSpace(fields[0])
		r.From = strings.TrimSpace(fields[2])
		r.To = strings.TrimSpace(fields[3])
		r.MsgNum = strings.TrimSpace(fields[6])
		r.Subject = strings.TrimSpace(fields[7])
		r.Folder = strings.TrimSpace(fields[8])

		// Timestamp is typically YYYY/MM/DD HH:MM in field[1].
		if ts := strings.TrimSpace(fields[1]); ts != "" {
			if t, err := time.Parse("2006/01/02 15:04", ts); err == nil {
				r.CreatedAt = t
			}
		}

		// Best-effort: find a state token like "Sent" in the remaining fields.
		for _, f := range fields {
			v := strings.TrimSpace(f)
			switch v {
			case "Sent", "Outbox", "Inbox", "Drafts", "Posted", "Received":
				r.State = v
			}
		}

		// Best-effort: frequency and GPS are often toward the end.
		if len(fields) >= 2 {
			last := strings.TrimSpace(fields[len(fields)-1])
			if strings.Contains(last, "GPS") || strings.Contains(last, "N,") || strings.Contains(last, "W") {
				r.GPS = last
			}
		}
		if len(fields) >= 3 {
			maybeFreq := strings.TrimSpace(fields[len(fields)-2])
			// In your examples this was "0" or "145090".
			if maybeFreq != "" {
				r.Freq = maybeFreq
			}
		}

		if r.ID != "" {
			out = append(out, r)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan Registry.txt: %w", err)
	}
	return out, nil
}
