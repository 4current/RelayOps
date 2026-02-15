package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/4current/relayops/internal/core"
	"github.com/4current/relayops/internal/runtime"
	"github.com/4current/relayops/internal/store"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("RelayOps %s\nCommit: %s\nBuilt: %s\n", version, commit, buildDate)

	case "doctor":
		runDoctor()

	case "init":
		runInit()

	case "compose":
		runCompose(os.Args[2:])

	case "list":
		runList(os.Args[2:])

	case "outbox":
		runOutbox(os.Args[2:])

	case "queue":
		runQueue(os.Args[2:])

	case "mark-sent":
		runMarkSent(os.Args[2:])

	case "mark-failed":
		runMarkFailed(os.Args[2:])

	case "send":
		runSend(os.Args[2:])

	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
	}
}

func printUsage() {
	fmt.Println("RelayOps - Radio Messaging Operations Engine")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  relayops version")
	fmt.Println("  relayops doctor")
	fmt.Println("  relayops init")
	fmt.Println("  relayops compose -s \"subject\" -b \"body\" [-t tag1,tag2] [-allow ...] [-prefer ...] [-session winlink|radio_only|post_office|p2p]")
	fmt.Println("  relayops list [-n 25]")
	fmt.Println("  relayops outbox [-n 25]")
	fmt.Println("  relayops queue -tag winlink_wednesday")
	fmt.Println("  relayops mark-sent -id <message-id>")
	fmt.Println("  relayops mark-failed -id <message-id> -err \"reason\"")
	fmt.Println("  relayops send [-tag t] [-n 25]  Send queued messages (simulated for now)")
	fmt.Println("")
}

func runDoctor() {
	fmt.Println("Running diagnostics...")

	dir, err := runtime.EnsureAppDir()
	if err != nil {
		fmt.Printf("✘ runtime dir: %v\n", err)
		return
	}
	fmt.Printf("✔ runtime dir: %s\n", dir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("✘ store open: %v\n", err)
		return
	}
	_ = st.Close()
	fmt.Println("✔ sqlite store: OK")

	testMsg := core.NewMessage("Test Subject", "Test Body")
	fmt.Printf("✔ message model OK (ID: %s)\n", testMsg.ID)

	fmt.Println("Diagnostics complete.")
}

func runInit() {
	dir, err := runtime.EnsureAppDir()
	if err != nil {
		fmt.Printf("Init failed: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("Init failed (store): %v\n", err)
		return
	}
	_ = st.Close()

	fmt.Printf("Initialized RelayOps runtime at: %s\n", dir)
}

func runCompose(args []string) {
	fs := flag.NewFlagSet("compose", flag.ContinueOnError)
	subject := fs.String("s", "", "subject")
	body := fs.String("b", "", "body")
	tagCSV := fs.String("t", "", "comma-separated tags")
	allowed := fs.String("allow", "", "allowed modes (comma-separated), e.g. packet,ardop,vara_hf")
	preferred := fs.String("prefer", "", "preferred modes (comma-separated), e.g. packet,vara_fm,telnet")
	session := fs.String("session", "winlink", "session mode: winlink, radio_only, post_office, p2p")
	_ = fs.Parse(args)

	if strings.TrimSpace(*subject) == "" || strings.TrimSpace(*body) == "" {
		fmt.Println("compose requires -s (subject) and -b (body)")
		fmt.Println("Example: relayops compose -s \"Winlink Wednesday\" -b \"Check-in\" -t winlink_wednesday")
		return
	}

	msg := core.NewMessage(*subject, *body)
	if strings.TrimSpace(*allowed) != "" {
		msg.Meta.Transport.Allowed = parseModes(*allowed)
	}

	if strings.TrimSpace(*preferred) != "" {
		msg.Meta.Transport.Preferred = parseModes(*preferred)
	}

	sess := core.SessionMode(strings.ToLower(strings.TrimSpace(*session)))
	switch sess {
	case core.SessionWinlink, core.SessionRadioOnly, core.SessionPostOffice, core.SessionP2P:
		msg.Meta.Session = sess
	default:
		fmt.Println("Invalid -session. Valid: winlink, radio_only, post_office, p2p")
		return
	}

	if strings.TrimSpace(*tagCSV) != "" {
		for _, t := range strings.Split(*tagCSV, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				msg.Tags = append(msg.Tags, t)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("store open failed: %v\n", err)
		return
	}
	defer func() { _ = st.Close() }()

	if err := st.SaveMessage(ctx, msg); err != nil {
		fmt.Printf("save failed: %v\n", err)
		return
	}

	fmt.Printf("Saved message: %s\n", msg.ID)
}

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	n := fs.Int("n", 25, "number of messages")
	_ = fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("store open failed: %v\n", err)
		return
	}
	defer func() { _ = st.Close() }()

	msgs, err := st.ListMessages(ctx, *n)
	if err != nil {
		fmt.Printf("list failed: %v\n", err)
		return
	}

	if len(msgs) == 0 {
		fmt.Println("(no messages)")
		return
	}

	for _, m := range msgs {
		ts := m.CreatedAt.Local().Format("2006-01-02 15:04:05")

		allow := modesToString(m.Meta.Transport.Allowed)
		prefer := modesToString(m.Meta.Transport.Preferred)
		sess := sessionToString(m.Meta.Session)

		// Build a compact metadata suffix
		var metaParts []string
		if sess != "" {
			metaParts = append(metaParts, "session="+sess)
		}
		if allow != "" {
			metaParts = append(metaParts, "allow="+allow)
		}
		if prefer != "" {
			metaParts = append(metaParts, "prefer="+prefer)
		}

		metaSuffix := ""
		if len(metaParts) > 0 {
			metaSuffix = "  " + strings.Join(metaParts, " ")
		}

		if len(m.Tags) > 0 {
			fmt.Printf("%s  %s  [%s]%s\n    %s\n", ts, m.ID, strings.Join(m.Tags, ","), metaSuffix, m.Subject)
		} else {
			fmt.Printf("%s  %s%s\n    %s\n", ts, m.ID, metaSuffix, m.Subject)
		}
	}
}

func parseModes(csv string) []core.Mode {
	var out []core.Mode
	for _, s := range strings.Split(csv, ",") {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			continue
		}
		out = append(out, core.Mode(s))
	}
	if len(out) == 0 {
		return []core.Mode{core.ModeAny}
	}
	return out
}

func modesToString(m []core.Mode) string {
	if len(m) == 0 {
		return ""
	}
	parts := make([]string, 0, len(m))
	for _, x := range m {
		parts = append(parts, string(x))
	}
	return strings.Join(parts, ",")
}

func sessionToString(s core.SessionMode) string {
	if s == "" {
		return string(core.SessionWinlink)
	}
	return string(s)
}

func runOutbox(args []string) {
	fs := flag.NewFlagSet("outbox", flag.ContinueOnError)
	n := fs.Int("n", 25, "number of messages")
	_ = fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("store open failed: %v\n", err)
		return
	}
	defer func() { _ = st.Close() }()

	msgs, err := st.ListByStatus(ctx, []core.MessageStatus{core.StatusDraft, core.StatusQueued, core.StatusFailed}, *n)
	if err != nil {
		fmt.Printf("outbox failed: %v\n", err)
		return
	}
	if len(msgs) == 0 {
		fmt.Println("(outbox empty)")
		return
	}

	for _, m := range msgs {
		ts := m.CreatedAt.Local().Format("2006-01-02 15:04:05")
		allow := modesToString(m.Meta.Transport.Allowed)
		prefer := modesToString(m.Meta.Transport.Preferred)
		sess := sessionToString(m.Meta.Session)

		fmt.Printf("%s  %s  status=%s  session=%s allow=%s prefer=%s\n    %s\n",
			ts,
			m.ID,
			m.Status,
			sess,
			allow,
			prefer,
			m.Subject,
		)
	}
}

func runQueue(args []string) {
	fs := flag.NewFlagSet("queue", flag.ContinueOnError)
	tag := fs.String("tag", "", "tag to queue (required)")
	_ = fs.Parse(args)

	if strings.TrimSpace(*tag) == "" {
		fmt.Println("queue requires -tag")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("store open failed: %v\n", err)
		return
	}
	defer func() { _ = st.Close() }()

	n, err := st.QueueByTag(ctx, *tag)
	if err != nil {
		fmt.Printf("queue failed: %v\n", err)
		return
	}
	fmt.Printf("Queued %d message(s) with tag: %s\n", n, *tag)
}

func runMarkSent(args []string) {
	fs := flag.NewFlagSet("mark-sent", flag.ContinueOnError)
	id := fs.String("id", "", "message id (required)")
	_ = fs.Parse(args)

	if strings.TrimSpace(*id) == "" {
		fmt.Println("mark-sent requires -id")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("store open failed: %v\n", err)
		return
	}
	defer func() { _ = st.Close() }()

	if err := st.SetStatusByID(ctx, *id, core.StatusSent, ""); err != nil {
		fmt.Printf("mark-sent failed: %v\n", err)
		return
	}
	fmt.Println("Marked sent:", *id)
}

func runMarkFailed(args []string) {
	fs := flag.NewFlagSet("mark-failed", flag.ContinueOnError)
	id := fs.String("id", "", "message id (required)")
	errMsg := fs.String("err", "send failed", "error message")
	_ = fs.Parse(args)

	if strings.TrimSpace(*id) == "" {
		fmt.Println("mark-failed requires -id")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("store open failed: %v\n", err)
		return
	}
	defer func() { _ = st.Close() }()

	if err := st.SetStatusByID(ctx, *id, core.StatusFailed, *errMsg); err != nil {
		fmt.Printf("mark-failed failed: %v\n", err)
		return
	}
	fmt.Println("Marked failed:", *id)
}

func runSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	tag := fs.String("tag", "", "only send queued messages with this tag")
	n := fs.Int("n", 25, "max messages to send")
	_ = fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	st, err := store.Open(ctx)
	if err != nil {
		fmt.Printf("store open failed: %v\n", err)
		return
	}
	defer func() { _ = st.Close() }()

	msgs, err := st.ListQueued(ctx, *tag, *n)
	if err != nil {
		fmt.Printf("send: list queued failed: %v\n", err)
		return
	}
	if len(msgs) == 0 {
		fmt.Println("(no queued messages)")
		return
	}

	// Sim transport for now (swap later with PAT)
	tx := NewSimTransport()

	var sent, failed int
	for _, m := range msgs {
		// Basic session sanity: P2P should not use telnet-only allow list
		if m.Meta.Session == core.SessionP2P && containsMode(m.Meta.Transport.Allowed, core.ModeTelnet) {
			_ = st.SetStatusByID(ctx, m.ID, core.StatusFailed, "session p2p incompatible with telnet allow-list")
			failed++
			continue
		}

		// Move to sending
		if err := st.MarkSending(ctx, m.ID); err != nil {
			fmt.Printf("mark sending failed (%s): %v\n", m.ID, err)
			failed++
			continue
		}

		// Send
		if err := tx.SendOne(ctx, m); err != nil {
			_ = st.SetStatusByID(ctx, m.ID, core.StatusFailed, err.Error())
			failed++
			continue
		}

		_ = st.SetStatusByID(ctx, m.ID, core.StatusSent, "")
		sent++
		fmt.Printf("Sent: %s  %q\n", m.ID, m.Subject)
	}

	fmt.Printf("Send complete. sent=%d failed=%d\n", sent, failed)
}

type SimTransport struct{}

func NewSimTransport() *SimTransport { return &SimTransport{} }

func (t *SimTransport) SendOne(ctx context.Context, m *core.Message) error {
	// Minimal “policy” use: if allowed list is empty, accept. If it contains "any", accept.
	// Otherwise accept as long as it doesn't explicitly forbid everything (we're simulating).
	if len(m.Meta.Transport.Allowed) == 0 || containsMode(m.Meta.Transport.Allowed, core.ModeAny) {
		return nil
	}
	// If someone explicitly set allowed only to telnet but session is radio_only, fail
	if m.Meta.Session == core.SessionRadioOnly && containsMode(m.Meta.Transport.Allowed, core.ModeTelnet) && len(m.Meta.Transport.Allowed) == 1 {
		return fmt.Errorf("radio_only session cannot be telnet-only")
	}
	return nil
}

func containsMode(list []core.Mode, x core.Mode) bool {
	for _, m := range list {
		if m == x {
			return true
		}
	}
	return false
}
