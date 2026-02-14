# RelayOps

RelayOps is a cross-platform **radio messaging operations engine**: a headless core + automation layer that makes Winlink-style messaging and field workflows reliable, repeatable, and mode-aware.

RelayOps is designed to be **Winlink-compatible**, while adding capabilities that traditional clients don’t prioritize:

- **Playbooks**: scripted operational workflows (e.g., Winlink Wednesday, weekly readiness checks)
- **Policies**: dynamic transport selection and fallback (VHF packet → HF soundcard → telnet)
- **Mode-aware messaging**: automatically adapt message formatting/attachments to the chosen RF mode
- **Audit logs**: deterministic “what happened” records for each session
- **Transport abstraction**: support multiple backends (starting with PAT, later more)

This project is intentionally built **engine-first**: the core runs headless and is testable from the CLI. A GUI shell can be added later without changing the protocol/automation logic.

---

## Why RelayOps?

Many ham messaging tools are Windows-first. Linux options exist, but the friction often shows up in:

- keeping forms updated
- workflow automation
- making the “right next choice” easy (transport, gateway, retries)
- pairing message content to RF constraints

RelayOps focuses on the missing layer: **operations**.

---

## Core Concepts

### Work Items
RelayOps is not “just email.” It treats tasks as *work items*:
- Winlink-compatible messages (store-and-forward)
- (future) packet BBS workflows
- (future) interactive sessions (chat, keyboard-to-keyboard)

### Message Metadata
RelayOps stores local-only metadata alongside messages:
- preferred transport order (e.g., VHF_PACKET, HF_ARDOP, TELNET)
- constraints (max airtime, attachment limits, plain-text-only on HF, etc.)
- tags for playbooks (e.g., `winlink_wednesday`)
- priority and retry strategy

This metadata is **not required** for compatibility and does not need to travel over Winlink.

### Policies
A policy evaluates message intent + system conditions and produces:
- a transport plan (ordered transports)
- transforms to apply (compact HF text, strip rich content, compress attachments)

### Playbooks
A playbook is an executable workflow, for example:

**Winlink Wednesday**
1. Generate message from template
2. Choose transport using policy + availability + historical success
3. Connect
4. Send tagged outbox items
5. Receive inbox
6. Log result + metrics

---

## Project Status

Early scaffold. Initial milestones:

1. Core message model + local store (SQLite)
2. CLI compose/send/receive with a test transport (telnet)
3. Transport abstraction + PAT backend integration
4. Policy engine (transport scoring + transforms)
5. Playbook runner + Winlink Wednesday playbook
6. GUI shell (optional)

---

## Development

### Requirements
- Go (current stable)
- SQLite (via pure-Go driver or CGO, TBD)

### Build
```bash
go build ./...

