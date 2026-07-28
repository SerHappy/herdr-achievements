package achievements

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func WithLockedState(dir string, fn func(*State) error) error {
	if dir == "" {
		return errors.New("HERDR_PLUGIN_STATE_DIR is not set")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	release, err := acquireLock(filepath.Join(dir, "state.lock"))
	if err != nil {
		return err
	}
	defer release()
	state, err := load(filepath.Join(dir, "state.json"))
	if err != nil {
		return err
	}
	if err := fn(&state); err != nil {
		return err
	}
	return save(filepath.Join(dir, "state.json"), state)
}

func LoadState(dir string) (State, error) { return load(filepath.Join(dir, "state.json")) }

func load(path string) (State, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewState(), nil
	}
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(b, &state); err != nil {
		return State{}, fmt.Errorf("state is corrupt: %w", err)
	}
	if state.Version != StateVersion {
		return State{}, fmt.Errorf("unsupported state version %d", state.Version)
	}
	if state.Unlocked == nil {
		state.Unlocked = map[string]string{}
	}
	if state.LastStatusByPane == nil {
		state.LastStatusByPane = map[string]string{}
	}
	return state, nil
}

func save(path string, state State) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err = tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	dirFile, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dirFile.Close()
	return dirFile.Sync()
}

func acquireLock(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}
