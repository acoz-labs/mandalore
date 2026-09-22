package readiness

// Observation contains measured identities, never values copied from a receipt
// merely because that receipt claims them. It is internal input, not a CLI field.
type Observation struct {
	Platform      Platform
	Identities    Identities
	ProcessSource string
}

type EvidenceMatch struct {
	EvidenceID string   `json:"evidence_id"`
	State      string   `json:"state"`
	Reasons    []string `json:"reasons"`
}

// matchEvidence answers only identity applicability for this recorded scenario.
// It does not test a live process or certify the rest of the machine.
func matchEvidence(e Evidence, observed Observation) EvidenceMatch {
	r := EvidenceMatch{EvidenceID: e.ID, State: "historical", Reasons: []string{}}
	if !validScenario(e.Component, e.Scenario) {
		r.Reasons = append(r.Reasons, "unknown-scenario")
		return r
	}
	if observed.Platform.OS == "" || observed.Platform.Arch == "" {
		r.Reasons = append(r.Reasons, "platform-unobserved")
	} else if e.Platform != observed.Platform {
		r.Reasons = append(r.Reasons, "platform-differs")
	}
	if observed.ProcessSource != "" && e.SourceCommit != "" && observed.ProcessSource != e.SourceCommit {
		r.Reasons = append(r.Reasons, "process-source-differs")
	}
	compare := func(key, expected, actual string) {
		if expected == "" {
			r.Reasons = append(r.Reasons, "record-"+key+"-missing")
		} else if actual == "" {
			r.Reasons = append(r.Reasons, key+"-unobserved")
		} else if expected != actual {
			r.Reasons = append(r.Reasons, key+"-differs")
		}
	}
	compare("runtime-sha256", e.Identities.RuntimeSHA256, observed.Identities.RuntimeSHA256)
	if e.Component == "codex" || e.Component == "pi" || e.Component == "claude-code" {
		compare("package-sha256", e.Identities.PackageSHA256, observed.Identities.PackageSHA256)
		compare("native-sha256", e.Identities.NativeSHA256, observed.Identities.NativeSHA256)
		compare("native-version", e.Identities.NativeVersion, observed.Identities.NativeVersion)
	}
	if e.Component == "pi" {
		compare("interpreter-sha256", e.Identities.InterpreterSHA256, observed.Identities.InterpreterSHA256)
		compare("interpreter-version", e.Identities.InterpreterVersion, observed.Identities.InterpreterVersion)
	}
	if len(r.Reasons) == 0 {
		r.State = "verified"
	}
	return r
}
