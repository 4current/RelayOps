package agent

import (
	"context"
)

type ServiceDef struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	ProcessName string   `json:"process_name"`
	Resources   []string `json:"resources"`
	TCPChecks   []string `json:"tcp_checks,omitempty"`
	FileChecks  []string `json:"file_checks,omitempty"`
	StartCmd    []string `json:"-"`
	StopCmd     []string `json:"-"`
}

var ServiceCatalog = map[string]ServiceDef{
	"rigctld-ts890": {
		Name:        "rigctld-ts890",
		Kind:        "cat",
		ProcessName: "rigctld",
		Resources:   []string{"ts890_cat"},
		TCPChecks:   []string{"127.0.0.1:4532"},
	},
	"ardop-ts890": {
		Name:        "ardop-ts890",
		Kind:        "modem",
		ProcessName: "ardopcf",
		Resources:   []string{"ts890_audio"},
		TCPChecks:   []string{"127.0.0.1:8515"},
		StartCmd:    []string{"start-ardop-ts890"},
		StopCmd:     []string{"stop-ardop-ts890"},
	},
	"varahf-ts890": {
		Name:        "varahf-ts890",
		Kind:        "modem",
		ProcessName: "VARA.exe",
		Resources:   []string{"ts890_audio"},
		TCPChecks:   []string{"127.0.0.1:8300"},
		StartCmd:    []string{"start-varahf-ts890"},
		StopCmd:     []string{"stop-varahf-ts890"},
	},
	"soundmodem-ic9700": {
		Name:        "soundmodem-ic9700",
		Kind:        "modem",
		ProcessName: "soundmodem",
		Resources:   []string{"ic9700_audio"},
		FileChecks:  []string{"/dev/soundmodem0"},
		StartCmd:    []string{"start-packet-ic9700"},
		StopCmd:     []string{"stop-packet-ic9700"},
	},
	"rigctld-ic9700": {
		Name:        "rigctld-ic9700",
		Kind:        "cat",
		ProcessName: "rigctld",
		Resources:   []string{"ic9700_cat"},
		TCPChecks:   []string{"127.0.0.1:4533"},
	},
	"varafm-ic9700": {
		Name:        "varafm-ic9700",
		Kind:        "modem",
		ProcessName: "VARAFM.exe",
		Resources:   []string{"ic9700_audio"},
		TCPChecks:   []string{"127.0.0.1:8300"},
		StartCmd:    []string{"start-varafm-ic9700"},
		StopCmd:     []string{"stop-varafm-ic9700"},
	},
}

type ServiceStatus struct {
	Name      string          `json:"name"`
	Kind      string          `json:"kind"`
	Running   bool            `json:"running"`
	Resources []string        `json:"resources"`
	TCP       map[string]bool `json:"tcp,omitempty"`
	Files     map[string]bool `json:"files,omitempty"`
}

func CollectServiceStatuses(ctx context.Context) map[string]ServiceStatus {
	out := make(map[string]ServiceStatus)

	for name, def := range ServiceCatalog {
		st := ServiceStatus{
			Name:      def.Name,
			Kind:      def.Kind,
			Running:   processExists(ctx, def.ProcessName),
			Resources: def.Resources,
			TCP:       map[string]bool{},
			Files:     map[string]bool{},
		}

		for _, addr := range def.TCPChecks {
			st.TCP[addr] = tcpOpen(addr)
		}
		for _, path := range def.FileChecks {
			st.Files[path] = fileExists(path)
		}

		out[name] = st
	}

	return out
}
