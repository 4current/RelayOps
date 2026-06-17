package agent

import (
	"context"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Status struct {
	Host      string          `json:"host"`
	Timestamp time.Time       `json:"timestamp"`
	Files     map[string]bool `json:"files"`
	TCP       map[string]bool `json:"tcp"`
	Processes map[string]bool `json:"processes"`
}

func CollectStatus(ctx context.Context) Status {
	host, _ := os.Hostname()

	return Status{
		Host:      host,
		Timestamp: time.Now(),
		Files: map[string]bool{
			"ts890_audio": exists("/dev/snd/by-radio/ts890-control"),
			"tty890A":     exists("/dev/tty890A"),
			"tty890B":     exists("/dev/tty890B"),
		},
		TCP: map[string]bool{
			"rigctld_ts890": tcpOpen("127.0.0.1:4532"),
			"ardopcf_ts890": tcpOpen("127.0.0.1:8515"),
		},
		Processes: map[string]bool{
			"rigctld":    processExists(ctx, "rigctld"),
			"ardopcf":    processExists(ctx, "ardopcf"),
			"soundmodem": processExists(ctx, "soundmodem"),
		},
	}
}

func exists(path string) bool {
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
