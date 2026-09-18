package memory

import (
	"strings"
	"testing"
)

func TestUpgradeRecordValidation(t *testing.T) {
	s := Signet{Version: 2, ID: "signet-test", Name: "Example"}
	valid := func() UpgradeRecord {
		return UpgradeRecord{Version: 1, ID: "upgrade-test", SignetID: s.ID, From: 1, To: 2, OriginalManifestSHA256: strings.Repeat("a", 64), BaseHead: strings.Repeat("b", 40), PortableSHA256: strings.Repeat("c", 64), RecordedAt: "2026-09-18T12:00:00Z", Authorship: Authorship{DeviceID: "device-test", Actor: "Example", Harness: "test"}}
	}
	device := func(string) error { return nil }
	if err := ValidateUpgradeRecords(s, []UpgradeRecord{valid()}, device); err != nil {
		t.Fatal(err)
	}
	if err := ValidateUpgradeRecords(s, nil, device); err == nil {
		t.Fatal("upgraded format lacks transition evidence")
	}
	for name, mutate := range map[string]func(*UpgradeRecord){
		"wrong identity":   func(u *UpgradeRecord) { u.SignetID = "signet-other" },
		"downgrade":        func(u *UpgradeRecord) { u.From, u.To = 2, 1 },
		"unsupported":      func(u *UpgradeRecord) { u.To = 3 },
		"missing base":     func(u *UpgradeRecord) { u.BaseHead = "" },
		"symbolic base":    func(u *UpgradeRecord) { u.BaseHead = "HEAD" },
		"manifest digest":  func(u *UpgradeRecord) { u.OriginalManifestSHA256 = strings.Repeat("z", 64) },
		"inventory digest": func(u *UpgradeRecord) { u.PortableSHA256 = "short" },
		"date":             func(u *UpgradeRecord) { u.RecordedAt = "now" },
		"author":           func(u *UpgradeRecord) { u.Authorship.Actor = "" },
	} {
		t.Run(name, func(t *testing.T) {
			u := valid()
			mutate(&u)
			if err := ValidateUpgradeRecords(s, []UpgradeRecord{u}, device); err == nil {
				t.Fatal("accepted invalid transition metadata")
			}
		})
	}
	first, second := valid(), valid()
	if err := ValidateUpgradeRecords(s, []UpgradeRecord{first, second}, device); err == nil {
		t.Fatal("duplicate receipt")
	}
	second.ID = "upgrade-other"
	if err := ValidateUpgradeRecords(s, []UpgradeRecord{first, second}, device); err != nil {
		t.Fatal("independent same-format upgrades cannot merge", err)
	}
	s.Version = 1
	if err := ValidateUpgradeRecords(s, []UpgradeRecord{first}, device); err == nil {
		t.Fatal("prepared upgrade treated as active old format")
	}
	if err := ValidateUpgradeRecords(s, nil, device); err != nil {
		t.Fatal("legacy format changed", err)
	}
}
