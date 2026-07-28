package achievements

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCorruptStateIsNotSilentlyOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	err := WithLockedState(dir, func(*State) error { return nil })
	if err == nil {
		t.Fatal("expected corrupt-state error")
	}
	b, readErr := os.ReadFile(path)
	if readErr != nil || string(b) != "not json" {
		t.Fatalf("state changed: %q, %v", b, readErr)
	}
}

func TestConcurrentStateTransactionsPreserveEveryUpdate(t *testing.T) {
	dir := t.TempDir()
	const writers = 48
	start := make(chan struct{})
	errs := make(chan error, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs <- WithLockedState(dir, func(state *State) error {
				// Keep the lock held long enough for simultaneous hook invocations to contend.
				time.Sleep(time.Millisecond)
				state.Unlocked[fmt.Sprintf("hook-%d", i)] = "now"
				return nil
			})
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	state, err := LoadState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Unlocked) != writers {
		t.Fatalf("lost updates: got %d, want %d", len(state.Unlocked), writers)
	}
	if _, err := os.ReadFile(filepath.Join(dir, "state.json")); err != nil {
		t.Fatalf("state.json is not readable: %v", err)
	}
}
