package achievements

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DecodeEvent reads Herdr's EventEnvelope and retains only the lifecycle fields
// needed by the reducer. Other payload fields, including workspace and agent
// metadata, are never retained.
func DecodeEvent(kind, raw string) (Event, bool, error) {
	switch kind {
	case "pane.agent_detected", "pane.agent_status_changed", "pane.closed", "pane.exited":
	default:
		return Event{}, false, nil
	}
	// Event hooks receive Herdr's EventEnvelope: {"event": ..., "data": {...}}.
	var envelope struct {
		Event string           `json:"event"`
		Data  *json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return Event{}, false, fmt.Errorf("invalid HERDR_PLUGIN_EVENT_JSON: %w", err)
	}
	schemaEvent := kindToSchemaEvent(kind)
	if envelope.Event != schemaEvent || envelope.Data == nil {
		return Event{}, false, nil
	}
	var payload struct {
		Type        string `json:"type"`
		PaneID      string `json:"pane_id"`
		WorkspaceID string `json:"workspace_id"`
		AgentStatus string `json:"agent_status"`
		Released    bool   `json:"released"`
	}
	if err := json.Unmarshal(*envelope.Data, &payload); err != nil {
		return Event{}, false, fmt.Errorf("invalid HERDR_PLUGIN_EVENT_JSON: %w", err)
	}
	if payload.Type != schemaEvent || payload.PaneID == "" || payload.WorkspaceID == "" {
		return Event{}, false, nil
	}
	if kind == "pane.agent_status_changed" && !validStatus(payload.AgentStatus) {
		return Event{}, false, nil
	}
	return Event{Kind: kind, PaneID: payload.PaneID, Status: payload.AgentStatus, Released: payload.Released}, true, nil
}

func kindToSchemaEvent(kind string) string {
	return strings.ReplaceAll(kind, ".", "_")
}

func validStatus(status string) bool {
	switch status {
	case "idle", "working", "blocked", "done", "unknown":
		return true
	}
	return false
}
