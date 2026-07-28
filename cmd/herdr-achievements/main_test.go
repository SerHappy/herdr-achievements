package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SerHappy/herdr-achievements/internal/achievements"
)

func TestRunEventNotifiesOnlyNewAchievement(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "notification.log")
	binPath := filepath.Join(dir, "herdr")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$NOTIFICATION_LOG\"\n"
	if err := os.WriteFile(binPath, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_BIN_PATH", binPath)
	t.Setenv("HERDR_PLUGIN_STATE_DIR", filepath.Join(dir, "state"))
	t.Setenv("HERDR_PLUGIN_EVENT", "pane.agent_detected")
	t.Setenv("HERDR_PLUGIN_EVENT_JSON", `{"type":"pane_agent_detected","pane_id":"w1:p1","workspace_id":"w1"}`)
	t.Setenv("NOTIFICATION_LOG", logPath)

	if err := runEvent(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "notification\nshow\n🏆 Achievement unlocked\n--body\nFIRST HOOF — Your first agent joined the herd.\n--sound\ndone\n"
	if string(got) != want {
		t.Fatalf("notification command = %q, want %q", got, want)
	}

	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}
	if err := runEvent(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("repeated event created notification log: %v", err)
	}
}

func TestRunEventKeepsSavedAchievementWhenNotificationFails(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "herdr")
	script := "#!/bin/sh\nexit 1\n"
	if err := os.WriteFile(binPath, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_BIN_PATH", binPath)
	stateDir := filepath.Join(dir, "state")
	t.Setenv("HERDR_PLUGIN_STATE_DIR", stateDir)
	t.Setenv("HERDR_PLUGIN_EVENT", "pane.agent_detected")
	t.Setenv("HERDR_PLUGIN_EVENT_JSON", `{"type":"pane_agent_detected","pane_id":"w1:p1","workspace_id":"w1"}`)

	var warnings bytes.Buffer
	previousStderr := stderr
	stderr = &warnings
	t.Cleanup(func() { stderr = previousStderr })

	if err := runEvent(); err != nil {
		t.Fatalf("runEvent returned notification failure: %v", err)
	}
	state, err := achievements.LoadState(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, unlocked := state.Unlocked[achievements.FirstHoof]; !unlocked {
		t.Fatal("achievement state was not saved before notification failure")
	}
	if !bytes.Contains(warnings.Bytes(), []byte("warning: could not show notification")) {
		t.Fatalf("stderr = %q, want notification warning", warnings.String())
	}
}

func TestRunEventReconciliationDropsStalePaneStatuses(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "herdr")
	script := "#!/bin/sh\nif [ \"$1\" = api ] && [ \"$2\" = snapshot ]; then printf '%s\\n' '{\"result\":{\"snapshot\":{\"agents\":[]}}}'; exit 0; fi\nexit 0\n"
	if err := os.WriteFile(binPath, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(dir, "state")
	if err := achievements.WithLockedState(stateDir, func(state *achievements.State) error {
		state.LastStatusByPane["old:p1"] = "working"
		state.PeakConcurrentWorking = 1
		state.Unlocked[achievements.FirstHoof] = "before-restart"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_BIN_PATH", binPath)
	t.Setenv("HERDR_PLUGIN_STATE_DIR", stateDir)
	if err := runReconcile(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_PLUGIN_EVENT", "pane.agent_status_changed")
	t.Setenv("HERDR_PLUGIN_EVENT_JSON", `{"type":"pane_agent_status_changed","pane_id":"new:p1","workspace_id":"w1","agent_status":"working"}`)

	if err := runEvent(); err != nil {
		t.Fatal(err)
	}
	state, err := achievements.LoadState(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, unlocked := state.Unlocked[achievements.DoubleTrouble]; unlocked {
		t.Fatal("stale working pane caused a false Double Trouble unlock")
	}
	if _, exists := state.LastStatusByPane["old:p1"]; exists {
		t.Fatal("stale pane status was retained after reconciliation")
	}
	if state.PeakConcurrentWorking != 1 {
		t.Fatalf("peak = %d, want preserved peak 1", state.PeakConcurrentWorking)
	}
	if _, unlocked := state.Unlocked[achievements.FirstHoof]; !unlocked {
		t.Fatal("reconciliation removed unlocked achievements")
	}
}

func TestRunEventReconciliationFailureOnlyClearsVolatileStatuses(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	if err := achievements.WithLockedState(stateDir, func(state *achievements.State) error {
		state.LastStatusByPane["stale-pane"] = "working"
		state.PeakConcurrentWorking = 3
		state.Unlocked[achievements.FullHerd] = "earlier"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_BIN_PATH", filepath.Join(dir, "missing-herdr"))
	t.Setenv("HERDR_PLUGIN_STATE_DIR", stateDir)
	if err := runReconcile(); err != nil {
		t.Fatal(err)
	}
	state, err := achievements.LoadState(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.LastStatusByPane) != 0 {
		t.Fatalf("volatile statuses = %v, want empty", state.LastStatusByPane)
	}
	if state.PeakConcurrentWorking != 3 {
		t.Fatalf("peak = %d, want 3", state.PeakConcurrentWorking)
	}
	if _, unlocked := state.Unlocked[achievements.FullHerd]; !unlocked {
		t.Fatal("reconciliation failure removed an unlocked achievement")
	}
}
