package achievements

import "testing"

// Fixtures use the exact required fields in the Herdr 0.7.5 installed schema.
func TestDecodeInstalledSchemaFixtures(t *testing.T) {
	tests := []struct {
		kind, json string
		want       Event
	}{
		{"pane.agent_detected", `{"event":"pane_agent_detected","data":{"type":"pane_agent_detected","pane_id":"w1:p1","workspace_id":"w1"}}`, Event{Kind: "pane.agent_detected", PaneID: "w1:p1"}},
		{"pane.agent_status_changed", `{"event":"pane_agent_status_changed","data":{"type":"pane_agent_status_changed","pane_id":"w1:p1","workspace_id":"w1","agent_status":"working"}}`, Event{Kind: "pane.agent_status_changed", PaneID: "w1:p1", Status: "working"}},
		{"pane.closed", `{"event":"pane_closed","data":{"type":"pane_closed","pane_id":"w1:p1","workspace_id":"w1"}}`, Event{Kind: "pane.closed", PaneID: "w1:p1"}},
		{"pane.exited", `{"event":"pane_exited","data":{"type":"pane_exited","pane_id":"w1:p1","workspace_id":"w1"}}`, Event{Kind: "pane.exited", PaneID: "w1:p1"}},
		{"pane.agent_detected", `{"event":"pane_agent_detected","data":{"type":"pane_agent_detected","pane_id":"w1:p1","workspace_id":"w1","released":true,"final_status":"done"}}`, Event{Kind: "pane.agent_detected", PaneID: "w1:p1", Released: true}},
	}
	for _, tt := range tests {
		got, ok, err := DecodeEvent(tt.kind, tt.json)
		if err != nil || !ok || got != tt.want {
			t.Fatalf("DecodeEvent(%s) = %#v, %v, %v", tt.kind, got, ok, err)
		}
	}
}

func TestDecodeIgnoresIncompleteAndUnknown(t *testing.T) {
	if _, ok, _ := DecodeEvent("pane.agent_status_changed", `{"event":"pane_agent_status_changed","data":{"type":"pane_agent_status_changed","pane_id":"p","workspace_id":"w"}}`); ok {
		t.Fatal("accepted missing status")
	}
	if _, ok, _ := DecodeEvent("pane.agent_detected", `{"event":"pane_agent_detected","data":{"type":"pane_agent_detected","pane_id":"p"}}`); ok {
		t.Fatal("accepted missing workspace")
	}
	if _, ok, _ := DecodeEvent("pane.agent_detected", `{"type":"pane_agent_detected","pane_id":"p","workspace_id":"w"}`); ok {
		t.Fatal("accepted direct payload")
	}
	if _, ok, _ := DecodeEvent("pane.agent_detected", `{"event":"pane_agent_detected","data":null}`); ok {
		t.Fatal("accepted null payload")
	}
	if _, ok, _ := DecodeEvent("pane_agent_detected", `{"event":"pane_agent_detected","data":{"type":"pane_agent_detected","pane_id":"p","workspace_id":"w"}}`); ok {
		t.Fatal("accepted schema spelling for event name")
	}
	if _, ok, _ := DecodeEvent("other", `{}`); ok {
		t.Fatal("accepted unknown event")
	}
}

func TestDecodeRejectsMismatchedEnvelope(t *testing.T) {
	if _, ok, err := DecodeEvent("pane.agent_detected", `{"event":"pane_agent_detected","data":{"type":"pane_agent_status_changed","pane_id":"p","workspace_id":"w","agent_status":"working"}}`); err != nil || ok {
		t.Fatalf("accepted mismatched payload type: ok=%v err=%v", ok, err)
	}
	if _, ok, err := DecodeEvent("pane.agent_detected", `{"event":"pane_agent_status_changed","data":{"type":"pane_agent_detected","pane_id":"p","workspace_id":"w"}}`); err != nil || ok {
		t.Fatalf("accepted mismatched envelope: ok=%v err=%v", ok, err)
	}
}
