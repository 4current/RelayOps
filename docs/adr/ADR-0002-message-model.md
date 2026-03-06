# ADR-0002: Internal message representation using MessageMeta

Date: 2026-02-16  
Status: Accepted

## Context

RelayOps must support multiple transport systems including:

- Winlink
- Pat
- potential future transports

A transport-independent message representation is required.

## Options Considered

1. Use Winlink message format internally
2. Use Pat B2F format internally
3. Define an internal message model

## Decision

Define an internal structure called MessageMeta.

Example:

MessageMeta
- Transport
- Session
- Constraints
- AutomationProfile

Transport adapters convert between MessageMeta and external formats.

## Consequences

### Positive

- transport abstraction
- extensibility

### Negative

- requires conversion logic for each transport