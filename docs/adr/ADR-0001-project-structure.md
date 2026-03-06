# ADR-0001: Use ADR documents for architectural decisions

Date: 2026-03-06  
Status: Accepted

## Context

RelayOps is being developed through exploratory engineering sessions.
Many technical decisions arise from experiments and ChatGPT discussions.

Without documentation, the reasoning behind decisions may be lost.

## Options Considered

1. No formal documentation
2. Maintain informal notes
3. Use Architecture Decision Records

## Decision

Adopt the ADR pattern to record architectural decisions.

Each decision will be documented as a Markdown file in docs/adr.

## Consequences

### Positive

- preserves reasoning behind decisions
- easy to review later
- integrates with Git history

### Negative

- small documentation overhead