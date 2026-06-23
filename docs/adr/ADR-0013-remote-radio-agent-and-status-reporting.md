# ADR-0013 Remote radio agent and status reporting

Date: 2026-06-17
Status: Proposed

---

## Context

RelayOps will be composed of aloose collecyion of tools running on different machines with networking between them. Need a management control center. RelayOps needs a reliable way to know which radio resources are present, busy, or available before starting mode-specific workflows.

---

## Options Considered

Option 1  
Option 2  
Option 3  

---

## Decision

Introduce a lightweight relayops-agent process on radio hosts such as Graviton.
Purpose: Expose read-only status for radios, audio devices, CAT daemons, TNC/modem processes, and TCP listeners.

Initial behavior: GET /status only.

Deferred: Start/stop control, scheduling, authentication, multi-host orchestration.


---

## Consequences

### Positive

- benefit
- benefit

### Negative

- tradeoff
- tradeoff

---

## Notes

Additional context, references, or links.