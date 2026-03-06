
# ADR-0004: Introduce transport abstraction layer
Date: 2026-03-06
Status: Accepted

## Context
RelayOps must support multiple transports.

## Decision
Create a transport interface with implementations such as:
- PatTransport
- TelnetTransport
- PacketTransport
