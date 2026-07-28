package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/SerHappy/herdr-achievements/internal/achievements"
	"github.com/SerHappy/herdr-achievements/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var stderr io.Writer = os.Stderr

const snapshotTimeout = 3 * time.Second

func main() {
	if len(os.Args) != 2 {
		fail("usage: herdr-achievements <event|reconcile|open|show>")
	}
	var err error
	switch os.Args[1] {
	case "event":
		err = runEvent()
	case "reconcile":
		err = runReconcile()
	case "open":
		err = runOpen()
	case "show":
		err = runShow()
	default:
		fail("unknown subcommand")
	}
	if err != nil {
		fail(err.Error())
	}
}

func runEvent() error {
	event, ok, err := achievements.DecodeEvent(os.Getenv("HERDR_PLUGIN_EVENT"), os.Getenv("HERDR_PLUGIN_EVENT_JSON"))
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	var unlocked []string
	err = achievements.WithLockedState(os.Getenv("HERDR_PLUGIN_STATE_DIR"), func(state *achievements.State) error {
		*state, unlocked = achievements.Reduce(*state, event, time.Now().UTC().Format(time.RFC3339Nano))
		return nil
	})
	if err != nil {
		return err
	}
	for _, id := range unlocked {
		achievement, ok := achievements.AchievementByID(id)
		if !ok {
			continue
		}
		if err := notify(achievement); err != nil {
			fmt.Fprintf(stderr, "herdr-achievements: warning: could not show notification for %s: %v\n", achievement.ID, err)
		}
	}
	return nil
}

func runReconcile() error {
	return achievements.WithLockedState(os.Getenv("HERDR_PLUGIN_STATE_DIR"), func(state *achievements.State) error {
		statuses, err := snapshotPaneStatuses(os.Getenv("HERDR_BIN_PATH"))
		if err != nil {
			fmt.Fprintf(stderr, "herdr-achievements: warning: could not reconcile pane statuses: %v; clearing volatile pane statuses\n", err)
			statuses = map[string]string{}
		}
		// Reconciliation intentionally replaces only ephemeral pane state.
		state.LastStatusByPane = statuses
		return nil
	})
}

func snapshotPaneStatuses(bin string) (map[string]string, error) {
	if bin == "" {
		return nil, fmt.Errorf("HERDR_BIN_PATH is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), snapshotTimeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, bin, "api", "snapshot").Output()
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(output, &envelope); err != nil || len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		if err != nil {
			return nil, fmt.Errorf("invalid snapshot JSON: %w", err)
		}
		return nil, fmt.Errorf("invalid snapshot JSON: missing result")
	}
	var result struct {
		Snapshot json.RawMessage `json:"snapshot"`
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil || len(result.Snapshot) == 0 || string(result.Snapshot) == "null" {
		if err != nil {
			return nil, fmt.Errorf("invalid snapshot JSON: %w", err)
		}
		return nil, fmt.Errorf("invalid snapshot JSON: missing result.snapshot")
	}
	var snapshot struct {
		Agents json.RawMessage `json:"agents"`
	}
	if err := json.Unmarshal(result.Snapshot, &snapshot); err != nil || len(snapshot.Agents) == 0 || string(snapshot.Agents) == "null" {
		if err != nil {
			return nil, fmt.Errorf("invalid snapshot JSON: %w", err)
		}
		return nil, fmt.Errorf("invalid snapshot JSON: missing result.snapshot.agents")
	}
	var agents []struct {
		PaneID      string `json:"pane_id"`
		AgentStatus string `json:"agent_status"`
	}
	if err := json.Unmarshal(snapshot.Agents, &agents); err != nil {
		return nil, fmt.Errorf("invalid snapshot JSON: %w", err)
	}
	statuses := make(map[string]string, len(agents))
	for _, agent := range agents {
		if agent.PaneID != "" && validAgentStatus(agent.AgentStatus) {
			statuses[agent.PaneID] = agent.AgentStatus
		}
	}
	return statuses, nil
}

func validAgentStatus(status string) bool {
	switch status {
	case "idle", "working", "blocked", "done", "unknown":
		return true
	}
	return false
}

func notify(achievement achievements.Achievement) error {
	bin := os.Getenv("HERDR_BIN_PATH")
	if bin == "" {
		return fmt.Errorf("HERDR_BIN_PATH is not set")
	}
	cmd := exec.Command(bin, "notification", "show", "🏆 Achievement unlocked", "--body", fmt.Sprintf("%s — %s", achievement.Name, achievement.Description), "--sound", "done")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func runOpen() error {
	bin := os.Getenv("HERDR_BIN_PATH")
	if bin == "" {
		return fmt.Errorf("HERDR_BIN_PATH is not set")
	}
	cmd := exec.Command(bin, "plugin", "pane", "open", "--plugin", "herdr-achievements", "--entrypoint", "achievements", "--placement", "popup", "--width", "88", "--height", "26", "--focus")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func runShow() error {
	state, err := achievements.LoadState(os.Getenv("HERDR_PLUGIN_STATE_DIR"))
	if err != nil {
		return err
	}
	result, err := tea.NewProgram(ui.NewModel(state), tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	model, ok := result.(ui.Model)
	if !ok {
		return fmt.Errorf("unexpected trophy room model %T", result)
	}
	seen := model.SeenIDs()
	if len(seen) == 0 {
		return nil
	}
	return persistSeen(os.Getenv("HERDR_PLUGIN_STATE_DIR"), seen)
}

// persistSeen performs a fresh locked read so achievements unlocked while the
// popup was open are retained.
func persistSeen(dir string, seen []string) error {
	return achievements.WithLockedState(dir, func(latest *achievements.State) error {
		if latest.Seen == nil {
			latest.Seen = map[string]bool{}
		}
		for _, id := range seen {
			if _, stillUnlocked := latest.Unlocked[id]; stillUnlocked {
				latest.Seen[id] = true
			}
		}
		return nil
	})
}
func fail(message string) { fmt.Fprintln(stderr, "herdr-achievements:", message); os.Exit(1) }
