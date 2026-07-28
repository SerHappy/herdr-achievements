package main

import (
	"os"
	"path/filepath"
	"testing"
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
