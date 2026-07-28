package achievements

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DecodeEvent deliberately reads only pane_id and agent_status. Other Herdr payload fields
// (including workspace and agent metadata) are never retained or used.
func DecodeEvent(kind, raw string) (Event, bool, error) {
	// Manifest subscriptions use dotted names, while the socket schema's
	// EventKind values use underscores. Normalize the runtime spelling.
	switch kind {
	case "pane_agent_detected":
		kind = "pane.agent_detected"
	case "pane_agent_status_changed":
		kind = "pane.agent_status_changed"
	}
	if kind != "pane.agent_detected" && kind != "pane.agent_status_changed" {
		return Event{}, false, nil
	}
	// Herdr's emitted EventEnvelope is {"event": ..., "data": {...}}. Keep
	// compatibility with a direct data object as well, since the decoder is
	// intentionally independent of the env transport.
	var envelope struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return Event{}, false, fmt.Errorf("invalid HERDR_PLUGIN_EVENT_JSON: %w", err)
	}
	payloadJSON := []byte(raw)
	if len(envelope.Data) != 0 && string(envelope.Data) != "null" {
		if envelope.Event != kindToSchemaEvent(kind) {
			return Event{}, false, nil
		}
		payloadJSON = envelope.Data
	}
	var payload struct {
		Type        string `json:"type"`
		PaneID      string `json:"pane_id"`
		WorkspaceID string `json:"workspace_id"`
		AgentStatus string `json:"agent_status"`
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return Event{}, false, fmt.Errorf("invalid HERDR_PLUGIN_EVENT_JSON: %w", err)
	}
	if payload.Type != kindToSchemaEvent(kind) || payload.PaneID == "" || payload.WorkspaceID == "" {
		return Event{}, false, nil
	}
	if kind == "pane.agent_status_changed" && !validStatus(payload.AgentStatus) {
		return Event{}, false, nil
	}
	return Event{Kind: kind, PaneID: payload.PaneID, Status: payload.AgentStatus}, true, nil
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
