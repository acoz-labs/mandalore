package memorycontext

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOrientationOnlyDoesNotAccessMemory(t *testing.T) {
	packet := Build(nil, "unused", false, Orientation)
	if packet.Context != Orientation || packet.Warning != "" {
		t.Fatal("orientation-only contract changed", packet)
	}
}

func TestEncodedBudgetIncludesEscapingAndDoesNotLeakOversizedContext(t *testing.T) {
	for _, value := range []string{strings.Repeat("x", MaxPacketBytes), strings.Repeat("\"", MaxPacketBytes/2), strings.Repeat("<", MaxPacketBytes/6)} {
		packet := Build(nil, "", false, value)
		raw, err := json.Marshal(packet)
		if err != nil || len(raw) > MaxPacketBytes || packet.Context != "" || packet.Warning == "" {
			t.Fatal("escaped packet exceeded budget or echoed oversized context", len(raw), err)
		}
	}
}
