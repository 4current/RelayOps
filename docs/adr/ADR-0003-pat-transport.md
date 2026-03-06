# ADR-0003: Use Pat CLI for Winlink transport

Date: 2026-02-18  
Status: Accepted

## Context

RelayOps requires automated message transmission through the Winlink system.

Winlink Express is primarily GUI-driven and difficult to automate.

Pat provides a command-line interface and supports multiple Winlink transports.

## Options Considered

1. Automate Winlink Express GUI
2. Reverse engineer Winlink protocol
3. Use Pat CLI
4. Implement a full Winlink stack

## Decision

Use Pat CLI as the primary transport mechanism.

RelayOps will generate message files and invoke Pat sessions.

## Consequences

### Positive

- automation-friendly
- cross-platform
- widely used in the ham community

### Negative

- requires Pat installation
- limited control over some internal features