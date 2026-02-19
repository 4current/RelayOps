package runtime

import (
	"os"
	"strings"
)

// IdentityScope returns a stable scope string used to namespace external message references.
//
// Scope is intended to represent the operator/mailbox identity across backends, not the transport.
// We use the format:
//   CALLSIGN            (when station is empty or "default")
//   CALLSIGN@STATION    (otherwise)
//
// Callsign is taken from the argument when provided, otherwise from RELAYOPS_CALLSIGN.
// Station is taken from RELAYOPS_STATION (default "default").
func IdentityScope(callsign string) string {
	cs := strings.ToUpper(strings.TrimSpace(callsign))
	if cs == "" {
		cs = strings.ToUpper(strings.TrimSpace(os.Getenv("RELAYOPS_CALLSIGN")))
	}

	station := strings.TrimSpace(os.Getenv("RELAYOPS_STATION"))
	if station == "" {
		station = "default"
	}

	// If we still don't have a callsign, return empty.
	if cs == "" {
		return ""
	}

	if station == "default" {
		return cs
	}
	return cs + "@" + station
}
