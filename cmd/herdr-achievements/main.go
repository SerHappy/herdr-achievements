package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/SerHappy/herdr-achievements/internal/achievements"
)

func main() {
	if len(os.Args) != 2 {
		fail("usage: herdr-achievements <event|open|show>")
	}
	var err error
	switch os.Args[1] {
	case "event":
		err = runEvent()
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
	err = achievements.WithLockedState(os.Getenv("HERDR_PLUGIN_STATE_DIR"), func(state *achievements.State) error {
		*state, _ = achievements.Reduce(*state, event, time.Now().UTC().Format(time.RFC3339Nano))
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func runOpen() error {
	bin := os.Getenv("HERDR_BIN_PATH")
	if bin == "" {
		return fmt.Errorf("HERDR_BIN_PATH is not set")
	}
	cmd := exec.Command(bin, "plugin", "pane", "open", "--plugin", "herdr-achievements", "--entrypoint", "achievements", "--placement", "popup", "--width", "72", "--height", "20", "--focus")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func runShow() error {
	state, err := achievements.LoadState(os.Getenv("HERDR_PLUGIN_STATE_DIR"))
	if err != nil {
		return err
	}
	fmt.Println("HERDR ACHIEVEMENTS")
	fmt.Printf("%d / %d unlocked\n\n", len(state.Unlocked), len(achievements.Catalog))
	for _, item := range achievements.Catalog {
		mark := "·"
		if _, ok := state.Unlocked[item.ID]; ok {
			mark = "✓"
		}
		line := fmt.Sprintf("%s %s", mark, item.Name)
		if item.Target > 0 {
			peak := min(state.PeakConcurrentWorking, item.Target)
			line += fmt.Sprintf("  %d / %d", peak, item.Target)
		}
		fmt.Println(line)
	}
	fmt.Println("\nPress Enter to close")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func fail(message string) { fmt.Fprintln(os.Stderr, "herdr-achievements:", message); os.Exit(1) }
