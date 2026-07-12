package agent

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Status struct {
	Host      string          `json:"host"`
	Timestamp time.Time       `json:"timestamp"`
	Facts     Facts           `json:"facts"`
	Services  ServiceStatuses `json:"services,omitempty"`
}

type Facts struct {
	Files      map[string]bool `json:"files"`
	TCP        map[string]bool `json:"tcp"`
	Processes  map[string]bool `json:"processes"`
	Interfaces map[string]bool `json:"interfaces"`
}

func CollectStatus(ctx context.Context) Status {

	host, _ := os.Hostname()
	facts := CollectFacts(ctx)
	return Status{
		Host:      host,
		Timestamp: time.Now(),
		Facts:     facts,
		Services:  EvaluateServices(facts),
	}

}

func soundmodemDevicePath() string {
	return fmt.Sprintf(
		"/run/user/%d/relayops/soundmodem0",
		os.Getuid(),
	)
}

func CollectFacts(ctx context.Context) Facts {

	return Facts{
		Files: map[string]bool{
			"ts890_audio":  fileExists("/dev/snd/by-radio/ts890-control"),
			"tty890A":      fileExists("/dev/tty890A"),
			"tty890B":      fileExists("/dev/tty890B"),
			"ic9700_audio": fileExists("/dev/snd/by-radio/ic9700-control"),
			"tty9700A":     fileExists("/dev/tty9700A"),
			"tty9700B":     fileExists("/dev/tty9700B"),
			"soundmodem0":  fileExists(soundmodemDevicePath()),
		},
		TCP: map[string]bool{
			"rigctld_ts890":  tcpOpen("127.0.0.1:4532"),
			"rigctld_ic9700": tcpOpen("127.0.0.1:4533"),
			"ardopcf_ts890":  tcpOpen("127.0.0.1:8515"),
			"varahf":         tcpOpen("127.0.0.1:8300"),
			"varafm":         tcpOpen("127.0.0.1:8301"),
		},
		Processes: map[string]bool{
			"rigctld":    processExists(ctx, "rigctld"),
			"ardopcf":    processExists(ctx, "ardopcf"),
			"soundmodem": processExists(ctx, "soundmodem"),
			"varahf":     processExistsPattern(ctx, "VARA.exe"),
			"varafm":     processExistsPattern(ctx, "VARAFM.exe"),
		},
		Interfaces: map[string]bool{
			"ax0": interfaceExists(ctx, "ax0"),
		}}

}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func tcpOpen(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func processExists(ctx context.Context, name string) bool {
	out, err := exec.CommandContext(ctx, "pgrep", "-x", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

func processExistsPattern(ctx context.Context, pattern string) bool {

	out, err := exec.CommandContext(ctx, "pgrep", "-f", pattern).Output()
	return err == nil && strings.TrimSpace(string(out)) != ""

}

func interfaceExists(ctx context.Context, name string) bool {
	err := exec.CommandContext(ctx, "ip", "link", "show", name).Run()
	return err == nil
}
