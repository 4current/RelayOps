
# ADR-0006: Filesystem-based outbound message queue
Date: 2026-03-06
Status: Proposed

## Context
Radio links are slow and intermittent.

## Decision
Use filesystem directories such as:

outbox/
sent/
failed/
