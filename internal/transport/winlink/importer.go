package winlink

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/runtime"
	"github.com/4current/relayops/internal/store"
	"github.com/google/uuid"
)

type ImportReport struct {
	Scanned int
	Created int
	Updated int
	Errors  int
}

// ImportFromWinlinkExpress imports (read-only) Winlink Express messages into the canonical RelayOps store.
//
// root must point at the callsign directory (e.g. .../RMS Express/AE4OK).
//
func ImportFromWinlinkExpress(ctx context.Context, st *store.Store, root string) (*ImportReport, error) {
	if st == nil {
		return nil, fmt.Errorf("ImportFromWinlinkExpress: store is nil")
	}
	recs, err := ReadRegistry(root)
	if err != nil {
		return nil, err
	}

	// Winlink Express roots are typically .../RMS Express/<CALLSIGN>. Use that as the mailbox identity for scope.
	// Station name is supplied via RELAYOPS_STATION (default "default").
	call := strings.ToUpper(strings.TrimSpace(filepath.Base(filepath.Clean(root))))
	scope := runtime.IdentityScope(call)

	report := &ImportReport{Scanned: len(recs)}
	for _, rec := range recs {
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		default:
		}

		messageID, found, err := st.GetMessageIDByExternalRef(ctx, "winlink", rec.ID, scope)
		if err != nil {
			report.Errors++
			continue
		}

		mimePath := filepath.Join(root, "Messages", rec.ID+".mime")
		raw, err := os.ReadFile(mimePath)
		if err != nil {
			report.Errors++
			continue
		}
		mimeHash := sha256.Sum256(raw)
		hashHex := hex.EncodeToString(mimeHash[:])

		parsed, body := parseRFC822(raw)
		// Prefer registry subject; fall back to MIME subject.
		subject := strings.TrimSpace(rec.Subject)
		if subject == "" {
			subject = strings.TrimSpace(parsed.Subject)
		}
		if subject == "" {
			subject = "(no subject)"
		}

		msg := &core.Message{}
		var created time.Time
		if !rec.CreatedAt.IsZero() {
			created = rec.CreatedAt
		} else if !parsed.Date.IsZero() {
			created = parsed.Date
		} else {
			created = time.Now()
		}

		if found {
			// For now we do not update the canonical message row (your Store lacks update/upsert).
			// We still keep backend state fresh.
			extra := buildBackendExtra(rec, hashHex)
			_ = st.UpsertBackendState(ctx, messageID, "winlink", rec.Folder, rec.State, extra)
			report.Updated++
			continue
		}

		// Create a new canonical message.
		msg.ID = uuid.NewString()
		msg.Subject = subject
		msg.Body = body
		msg.CreatedAt = created
		msg.UpdatedAt = created
		msg.LastError = ""
		msg.Tags = []string{}
		msg.Meta = core.DefaultMeta()
		msg.Meta.Session = core.SessionWinlink
		msg.Meta.Transport.Allowed = []core.Mode{core.ModeAny}
		msg.Meta.Transport.Preferred = []core.Mode{}

		// Addresses: prefer MIME headers, but fall back to Registry.
		msg.From = core.Address{Callsign: strings.TrimSpace(rec.From)}
		if parsed.FromCallsign != "" {
			msg.From = core.Address{Callsign: parsed.FromCallsign, Email: parsed.FromEmail}
		}
		msg.To = parsed.To
		if len(msg.To) == 0 {
			to := strings.TrimSpace(rec.To)
			if to != "" {
				msg.To = []core.Address{{Callsign: to}}
			}
		}

		// Map folder/state into RelayOps-local status (coarse).
		msg.Status = mapWinlinkFolderToStatus(rec.Folder)
		if rec.State == "Sent" {
			t := created
			msg.SentAt = &t
			msg.Status = core.StatusSent
		}

		if err := st.SaveMessage(ctx, msg); err != nil {
			report.Errors++
			continue
		}

		// Write external ref.
		if err := st.UpsertExternalRef(ctx, msg.ID, "winlink", rec.ID, scope, "{}"); err != nil {
			report.Errors++
			// message row exists; keep going
		}

		// Store backend state (folder/status + extras).
		extra := buildBackendExtra(rec, hashHex)
		_ = st.UpsertBackendState(ctx, msg.ID, "winlink", rec.Folder, rec.State, extra)

		report.Created++
	}

	return report, nil
}

type parsedHeaders struct {
	Subject      string
	Date         time.Time
	FromCallsign string
	FromEmail    string
	To           []core.Address
}

func parseRFC822(raw []byte) (parsedHeaders, string) {
	var out parsedHeaders
	body := ""

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return out, body
	}
	out.Subject = msg.Header.Get("Subject")
	if ds := msg.Header.Get("Date"); ds != "" {
		if t, err := mail.ParseDate(ds); err == nil {
			out.Date = t
		}
	}

	// From
	if f := msg.Header.Get("From"); f != "" {
		if addrs, err := mail.ParseAddressList(f); err == nil && len(addrs) > 0 {
			// Winlink From often looks like CALLSIGN@winlink.org
			out.FromEmail = addrs[0].Address
			out.FromCallsign = strings.Split(addrs[0].Address, "@")[0]
		}
	}

	// To
	if t := msg.Header.Get("To"); t != "" {
		if addrs, err := mail.ParseAddressList(t); err == nil {
			for _, a := range addrs {
				addr := core.Address{}
				if strings.Contains(a.Address, "@") {
					addr.Email = a.Address
					addr.Callsign = strings.Split(a.Address, "@")[0]
				} else {
					addr.Callsign = a.Address
				}
				out.To = append(out.To, addr)
			}
		}
	}

	// Body: best-effort read raw body (may include MIME multipart).
	b, _ := io.ReadAll(msg.Body)
	body = strings.TrimSpace(string(b))

	return out, body
}

func mapWinlinkFolderToStatus(folder string) core.MessageStatus {
	f := strings.ToLower(strings.TrimSpace(folder))
	switch f {
	case "sent items", "sent":
		return core.StatusSent
	case "outbox":
		return core.StatusQueued
	default:
		// inbox/drafts/etc: keep as draft until you add a dedicated received status.
		return core.StatusDraft
	}
}

func buildBackendExtra(rec RegistryRecord, mimeSHA256 string) string {
	extra := map[string]any{
		"msg_num":     rec.MsgNum,
		"subject":     rec.Subject,
		"folder":      rec.Folder,
		"state":       rec.State,
		"freq":        rec.Freq,
		"gps":         rec.GPS,
		"mime_sha256": mimeSHA256,
	}
	if !rec.CreatedAt.IsZero() {
		extra["created_at"] = rec.CreatedAt.UTC().Format(time.RFC3339)
	}
	if len(rec.RawFields) > 0 {
		// Keep a slim record of unknown fields for future reverse engineering.
		extra["raw_field_count"] = len(rec.RawFields)
	}
	b, _ := json.Marshal(extra)
	return string(b)
}
