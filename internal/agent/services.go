package agent

type ServiceDef struct {
	Name string   `json:"name"`
	Kind string   `json:"kind"`
	Uses []string `json:"uses"`

	Checks   []CheckRef `json:"checks"`
	StartCmd []string   `json:"-"`
	StopCmd  []string   `json:"-"`
}

type CheckRef struct {
	Kind string `json:"kind"` // file, tcp, process
	Key  string `json:"key"`
}

type ServiceStatus struct {
	Name   string          `json:"name"`
	Kind   string          `json:"kind"`
	Uses   []string        `json:"uses"`
	Ready  bool            `json:"ready"`
	Checks map[string]bool `json:"checks"`
}

type ServiceStatuses map[string]ServiceStatus

var ServiceCatalog = map[string]ServiceDef{
	"rigctld-ts890": {
		Name: "rigctld-ts890",
		Kind: "cat",
		Uses: []string{"ts890_cat"},
		Checks: []CheckRef{
			{Kind: "tcp", Key: "rigctld_ts890"},
		},
		StartCmd: []string{"start-rigctl-ts890"},
		StopCmd:  []string{"stop-rigctl-ts890"},
	},

	"ardop-ts890": {
		Name: "ardop-ts890",
		Kind: "modem",
		Uses: []string{"ts890_audio"},
		Checks: []CheckRef{
			{Kind: "process", Key: "ardopcf"},
			{Kind: "tcp", Key: "ardopcf_ts890"},
			{Kind: "file", Key: "ts890_audio"},
		},
		StartCmd: []string{"start-ardop-ts890"},
		StopCmd:  []string{"stop-ardop-ts890"},
	},

	"varahf-ts890": {
		Name: "varahf-ts890",
		Kind: "modem",
		Uses: []string{"ts890_audio"},
		Checks: []CheckRef{
			{Kind: "process", Key: "varahf"},
			{Kind: "tcp", Key: "varahf"},
		},
		StartCmd: []string{"start-varahf-ts890"},
		StopCmd:  []string{"stop-varahf-ts890"},
	},

	"rigctld-ic9700": {
		Name: "rigctld-ic9700",
		Kind: "cat",
		Uses: []string{"ic9700_cat"},
		Checks: []CheckRef{
			{Kind: "tcp", Key: "rigctld_ic9700"},
		},
		StartCmd: []string{"start-rigctl-ic9700"},
		StopCmd:  []string{"stop-rigctl-ic9700"},
	},

	"soundmodem-ic9700": {
		Name: "soundmodem-ic9700",
		Kind: "modem",
		Uses: []string{"ic9700_audio"},
		Checks: []CheckRef{
			{Kind: "process", Key: "soundmodem"},
			{Kind: "file", Key: "soundmodem0"},
			{Kind: "interface", Key: "ax0"},
		},
		StartCmd: []string{"start-packet-ic9700"},
		StopCmd:  []string{"stop-packet-ic9700"},
	},

	"varafm-ic9700": {
		Name: "varafm-ic9700",
		Kind: "modem",
		Uses: []string{"ic9700_audio"},
		Checks: []CheckRef{
			{Kind: "process", Key: "varafm"},
			{Kind: "tcp", Key: "varafm"},
		},
		StartCmd: []string{"start-varafm-ic9700"},
		StopCmd:  []string{"stop-varafm-ic9700"},
	},
}

func EvaluateServices(facts Facts) ServiceStatuses {
	out := make(ServiceStatuses)

	for id, def := range ServiceCatalog {
		checks := make(map[string]bool)
		ready := true

		for _, chk := range def.Checks {
			label := chk.Kind + ":" + chk.Key
			ok := checkFact(facts, chk)
			checks[label] = ok
			if !ok {
				ready = false
			}
		}

		out[id] = ServiceStatus{
			Name:   def.Name,
			Kind:   def.Kind,
			Uses:   def.Uses,
			Ready:  ready,
			Checks: checks,
		}
	}

	return out
}

func checkFact(facts Facts, chk CheckRef) bool {
	switch chk.Kind {
	case "file":
		return facts.Files[chk.Key]
	case "tcp":
		return facts.TCP[chk.Key]
	case "process":
		return facts.Processes[chk.Key]
	case "interface":
		return facts.Interfaces[chk.Key]
	default:
		return false
	}
}
