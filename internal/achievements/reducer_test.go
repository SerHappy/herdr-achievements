package achievements

import "testing"

func TestAchievements(t *testing.T) {
	tests := []struct {
		name   string
		events []Event
		want   []string
	}{
		{"first hoof", []Event{{Kind: "pane.agent_detected", PaneID: "p1"}}, []string{FirstHoof}},
		{"first delivery idle", []Event{{Kind: "pane.agent_status_changed", PaneID: "p1", Status: "working"}, {Kind: "pane.agent_status_changed", PaneID: "p1", Status: "idle"}}, []string{FirstDelivery}},
		{"first delivery done", []Event{{Kind: "pane.agent_status_changed", PaneID: "p1", Status: "working"}, {Kind: "pane.agent_status_changed", PaneID: "p1", Status: "done"}}, []string{FirstDelivery}},
		{"unstuck", []Event{{Kind: "pane.agent_status_changed", PaneID: "p1", Status: "blocked"}, {Kind: "pane.agent_status_changed", PaneID: "p1", Status: "working"}}, []string{Unstuck}},
		{"double trouble", []Event{{Kind: "pane.agent_status_changed", PaneID: "p1", Status: "working"}, {Kind: "pane.agent_status_changed", PaneID: "p2", Status: "working"}}, []string{DoubleTrouble}},
		{"full herd", []Event{{Kind: "pane.agent_status_changed", PaneID: "p1", Status: "working"}, {Kind: "pane.agent_status_changed", PaneID: "p2", Status: "working"}, {Kind: "pane.agent_status_changed", PaneID: "p3", Status: "working"}}, []string{DoubleTrouble, FullHerd}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewState()
			got := map[string]bool{}
			for _, event := range tt.events {
				var unlocked []string
				state, unlocked = Reduce(state, event, "2026-01-01T00:00:00Z")
				for _, id := range unlocked {
					got[id] = true
				}
			}
			for _, id := range tt.want {
				if !got[id] {
					t.Errorf("missing %s", id)
				}
			}
		})
	}
}

func TestRepeatedEventDoesNotUnlockTwice(t *testing.T) {
	state := NewState()
	state, first := Reduce(state, Event{Kind: "pane.agent_detected", PaneID: "p1"}, "first")
	_, second := Reduce(state, Event{Kind: "pane.agent_detected", PaneID: "p1"}, "second")
	if len(first) != 1 || len(second) != 0 {
		t.Fatalf("got first=%v second=%v", first, second)
	}
}

func TestConcurrencyAcrossPanes(t *testing.T) {
	state := NewState()
	for _, event := range []Event{{Kind: "pane.agent_status_changed", PaneID: "one", Status: "working"}, {Kind: "pane.agent_status_changed", PaneID: "two", Status: "working"}, {Kind: "pane.agent_status_changed", PaneID: "one", Status: "idle"}} {
		state, _ = Reduce(state, event, "now")
	}
	if state.PeakConcurrentWorking != 2 {
		t.Fatalf("peak = %d", state.PeakConcurrentWorking)
	}
	if state.LastStatusByPane["two"] != "working" {
		t.Fatal("other pane status was lost")
	}
}

func TestClosedOrExitedPaneIsRemoved(t *testing.T) {
	for _, kind := range []string{"pane.closed", "pane.exited"} {
		t.Run(kind, func(t *testing.T) {
			state := NewState()
			state.LastStatusByPane["p1"] = "working"
			state, unlocked := Reduce(state, Event{Kind: kind, PaneID: "p1"}, "now")
			if len(unlocked) != 0 {
				t.Fatalf("unexpected unlocks: %v", unlocked)
			}
			if _, exists := state.LastStatusByPane["p1"]; exists {
				t.Fatal("closed pane status was retained")
			}
		})
	}
}

func TestReleasedAgentDetectionRemovesPaneWithoutUnlocking(t *testing.T) {
	state := NewState()
	state.LastStatusByPane["p1"] = "working"
	state, unlocked := Reduce(state, Event{Kind: "pane.agent_detected", PaneID: "p1", Released: true}, "now")
	if len(unlocked) != 0 {
		t.Fatalf("unexpected unlocks: %v", unlocked)
	}
	if _, exists := state.LastStatusByPane["p1"]; exists {
		t.Fatal("released pane status was retained")
	}
	if _, unlocked := state.Unlocked[FirstHoof]; unlocked {
		t.Fatal("released detection unlocked First Hoof")
	}
}
