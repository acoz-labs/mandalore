package memory

import "testing"

func TestConnectionMetadataValidatesOnlySuppliedIdentity(t *testing.T) {
	signet := Signet{Version: FormatVersion, ID: "signet-synthetic", Name: "Synthetic bank"}
	device := Device{Version: 1, ID: "device-synthetic", Label: "Synthetic machine"}
	author := Authorship{DeviceID: device.ID, Actor: "Synthetic operator", Harness: "pi"}
	if err := ValidateConnectionMetadata(signet, device, author); err != nil {
		t.Fatal(err)
	}
	upgraded := signet
	upgraded.Version = 2
	if err := ValidateConnectionMetadata(upgraded, device, author); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Signet, *Device, *Authorship){
		func(s *Signet, _ *Device, _ *Authorship) { s.Version = 3 },
		func(s *Signet, _ *Device, _ *Authorship) { s.ID = "../wrong" },
		func(s *Signet, _ *Device, _ *Authorship) { s.Name = "two\nlines" },
		func(_ *Signet, d *Device, _ *Authorship) { d.Version = 2 },
		func(_ *Signet, d *Device, _ *Authorship) { d.Label = " " },
		func(_ *Signet, d *Device, _ *Authorship) { d.ID = "device-other" },
		func(_ *Signet, _ *Device, a *Authorship) { a.DeviceID = "../wrong" },
		func(_ *Signet, _ *Device, a *Authorship) { a.Actor = " " },
		func(_ *Signet, _ *Device, a *Authorship) { a.Harness = " " },
	} {
		s, d, a := signet, device, author
		mutate(&s, &d, &a)
		if err := ValidateConnectionMetadata(s, d, a); err == nil {
			t.Fatal("accepted invalid connection metadata")
		}
	}
}
