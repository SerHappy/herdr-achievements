package achievements

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCurrentReleaseStateStillLoads(t *testing.T) {
	dir := t.TempDir()
	currentRelease := []byte(`{
  "version": 1,
  "unlocked": {"first-hoof": "2026-01-01T00:00:00Z"},
  "last_status_by_pane": {"pane-1": "working"},
  "peak_concurrent_working": 1
}
`)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), currentRelease, 0600); err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if state.Seen == nil {
		t.Fatal("seen was not initialized for a current-release state file")
	}
	if state.LastStatusByPane["pane-1"] != "working" || state.PeakConcurrentWorking != 1 {
		t.Fatalf("existing state changed while loading: %#v", state)
	}
}

func TestUnseenUnlockedIsDeterministic(t *testing.T) {
	state := NewState()
	state.Unlocked[FullHerd] = "later"
	state.Unlocked[FirstHoof] = "first"
	state.Unlocked[Unstuck] = "middle"
	state.Seen[Unstuck] = true
	want := []string{FirstHoof, FullHerd}
	for range 20 {
		if got := UnseenUnlocked(state); !reflect.DeepEqual(got, want) {
			t.Fatalf("UnseenUnlocked() = %v, want %v", got, want)
		}
	}
}
