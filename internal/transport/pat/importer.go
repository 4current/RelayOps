package pat

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/store"
	"github.com/google/uuid"
)

// ImportReport mirrors the winlink importer report.
type ImportReport struct {
	Scanned int
	Created int
	Updated int
	Errors  int
}

// ImportFromMailbox imports (read-only) PAT mailbox messages into the canonical RelayOps store.
//
// It scans: <mbox>/<CALLSIGN>/{in,out,sent,archive}/*.b2f
// and uses `pat extract <MID>` to decode each message.
//
// callsign is optional; if empty RELAYOPS_CALLSIGN is used.
func ImportFromMailbox(ctx context.Context, st *store.Store, patBinary, mbox, callsign string, scope string) (*ImportReport, error) {
	if st == nil {
		return nil, fmt.Errorf("ImportFromMailbox: store is nil")
	}
	if strings.TrimSpace(patBinary) == "" {
		patBinary = "pat"
	}
	call := strings.ToUpper(strings.TrimSpace(callsign))
	if call == "" {
		call = strings.ToUpper(strings.TrimSpace(os.Getenv("RELAYOPS_CALLSIGN")))
	}
	if call == "" {
		return nil, fmt.Errorf("ImportFromMailbox: callsign is required (set RELAYOPS_CALLSIGN or pass -callsign)")
	}
	if strings.TrimSpace(mbox) == "" {
		return nil, fmt.Errorf("ImportFromMailbox: mbox path is required")
	}

	if strings.TrimSpace(scope) == "" {
		return nil, fmt.Errorf("ImportFromMailbox: scope is required")
	}

	folders := []string{"in", "out", "sent", "archive"}
	var files []string
	for _, f := range folders {
		glob := filepath.Join(mbox, call, f, "*.b2f")
		matches, _ := filepath.Glob(glob)
		files = append(files, matches...)
	}

	report := &ImportReport{Scanned: len(files)}
	for _, p := range files {
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		default:
		}

		mid := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		if mid == "" {
			report.Errors++
			continue
		}

		folderKey := strings.ToLower(filepath.Base(filepath.Dir(p)))
		folder, state, canonicalStatus := mapPatFolder(folderKey)

		messageID, found, err := st.GetMessageIDByExternalRef(ctx, "pat", mid, scope)
		if err != nil {
			report.Errors++
			continue
		}

		hdr, body, extra, err := patExtract(ctx, patBinary, mbox, call, mid)
		if err != nil {
			report.Errors++
			continue
		}
		extra.Folder = folder
		extra.Scope = scope
		extraJSON, _ := json.Marshal(extra)

		if found {
			if err := st.UpsertBackendState(ctx, messageID, "pat", folder, state, string(extraJSON)); err != nil {
				report.Errors++
			} else {
				report.Updated++
			}
			continue
		}

		now := time.Now()
		created := hdr.Date
		if created.IsZero() {
			created = now
		}

		subj := strings.TrimSpace(hdr.Subject)
		if subj == "" {
			subj = "(no subject)"
		}

		msg := &core.Message{
			ID:        uuid.NewString(),
			Subject:   subj,
			Body:      body,
			From:      core.Address{Callsign: hdr.From},
			To:        hdr.To,
			Tags:      []string{},
			CreatedAt: created,
			UpdatedAt: now,
			Status:    canonicalStatus,
			Meta:      core.DefaultMeta(),
			LastError: "",
		}

		if err := st.SaveMessage(ctx, msg); err != nil {
			report.Errors++
			continue
		}

		if err := st.UpsertExternalRef(ctx, msg.ID, "pat", mid, scope, "{}"); err != nil {
			report.Errors++
			continue
		}

		if err := st.UpsertBackendState(ctx, msg.ID, "pat", folder, state, string(extraJSON)); err != nil {
			report.Errors++
			continue
		}

		report.Created++
	}

	return report, nil
}

type patHeader struct {
	MID     string
	Date    time.Time
	From    string
	To      []core.Address
	Subject string
}

type patExtra struct {
	MID        string `json:"mid"`
	Folder     string `json:"folder"`
	Scope      string `json:"scope"`
	TextHash   string `json:"text_sha256"`
	RawDate    string `json:"raw_date"`
	RawSubject string `json:"raw_subject"`
}

func patExtract(ctx context.Context, patBinary, mbox, call, mid string) (*patHeader, string, *patExtra, error) {
	args := []string{"--mbox", mbox, "--mycall", call, "extract", mid}
	cmd := exec.CommandContext(ctx, patBinary, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, "", nil, fmt.Errorf("pat extract failed: %w: %s", err, strings.TrimSpace(out.String()))
	}
	text := out.String()
	hdr, body := parsePatDump(text)

	sum := sha256.Sum256([]byte(text))
	hexHash := hex.EncodeToString(sum[:])
	extra := &patExtra{MID: hdr.MID, TextHash: hexHash, RawDate: hdr.Date.Format(time.RFC3339), RawSubject: hdr.Subject}
	return hdr, body, extra, nil
}

func parsePatDump(s string) (*patHeader, string) {
	h := &patHeader{}
	lines := strings.Split(s, "\n")
	i := 0
	for ; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		if strings.TrimSpace(line) == "" {
			i++
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch strings.ToLower(key) {
		case "mid":
			h.MID = val
		case "date":
			if t, err := time.Parse("2006-01-02 15:04:05 -0700 MST", val); err == nil {
				h.Date = t
			} else if t, err := time.Parse(time.RFC3339, val); err == nil {
				h.Date = t
			}
		case "from":
			h.From = val
		case "to":
			if val != "" {
				if strings.Contains(val, "@") {
					h.To = append(h.To, core.Address{Email: val})
				} else {
					h.To = append(h.To, core.Address{Callsign: val})
				}
			}
		case "subject":
			h.Subject = val
		}
	}
	body := strings.TrimRight(strings.Join(lines[i:], "\n"), "\n")
	return h, body
}

func mapPatFolder(folderKey string) (folder string, state string, canonical core.MessageStatus) {
	switch folderKey {
	case "in":
		return "InBox", "Received", core.StatusDraft
	case "out":
		return "Outbox", "Queued", core.StatusQueued
	case "sent":
		return "Sent", "Sent", core.StatusSent
	case "archive":
		return "Archive", "Archived", core.StatusDraft
	default:
		return folderKey, "", core.StatusDraft
	}
}
