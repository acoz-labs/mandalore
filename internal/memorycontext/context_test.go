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

func TestOrdinaryFreshnessGuidanceDoesNotNeedRemoteCue(t *testing.T) {
	packet := Build(nil, "", false, Orientation)
	for _, required := range []string{"At the start of ordinary memory work", "current task and connection permit synchronization", "Do not wait for a cross-machine cue", "Do not repeat for every lookup", "synchronization is prohibited or unavailable"} {
		if !strings.Contains(packet.Context, required) {
			t.Fatalf("shared orientation omits %q", required)
		}
	}
}
