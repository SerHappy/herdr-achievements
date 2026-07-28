// Package achievements contains the privacy-preserving achievement reducer.
package achievements

import "sort"

const StateVersion = 1

const (
	FirstHoof     = "first-hoof"
	FirstDelivery = "first-delivery"
	Unstuck       = "unstuck"
	DoubleTrouble = "double-trouble"
	FullHerd      = "full-herd"
)

type Achievement struct {
	ID          string
	Name        string
	Description string
	Target      int
}

var Catalog = []Achievement{
	{FirstHoof, "FIRST HOOF", "Your first agent joined the herd.", 0},
	{FirstDelivery, "FIRST DELIVERY", "An agent finished a stretch of work.", 0},
	{Unstuck, "UNSTUCK", "An agent got back to work.", 0},
	{DoubleTrouble, "DOUBLE TROUBLE", "Two agents were working at once.", 2},
	{FullHerd, "FULL HERD", "Three agents were working at once.", 3},
}

type State struct {
	Version               int               `json:"version"`
	Unlocked              map[string]string `json:"unlocked"`
	LastStatusByPane      map[string]string `json:"last_status_by_pane"`
	PeakConcurrentWorking int               `json:"peak_concurrent_working"`
}

type Event struct {
	Kind   string
	PaneID string
	Status string
}

func NewState() State {
	return State{Version: StateVersion, Unlocked: map[string]string{}, LastStatusByPane: map[string]string{}}
}

// Reduce is deterministic: callers provide the timestamp string used for newly unlocked items.
func Reduce(state State, event Event, nowUTC string) (State, []string) {
	if state.Version == 0 {
		state.Version = StateVersion
	}
	if state.Unlocked == nil {
		state.Unlocked = map[string]string{}
	}
	if state.LastStatusByPane == nil {
		state.LastStatusByPane = map[string]string{}
	}
	var unlocked []string
	unlock := func(id string) {
		if _, exists := state.Unlocked[id]; !exists {
			state.Unlocked[id] = nowUTC
			unlocked = append(unlocked, id)
		}
	}

	switch event.Kind {
	case "pane.agent_detected":
		unlock(FirstHoof)
	case "pane.agent_status_changed":
		previous := state.LastStatusByPane[event.PaneID]
		state.LastStatusByPane[event.PaneID] = event.Status
		if previous == "working" && (event.Status == "done" || event.Status == "idle") {
			unlock(FirstDelivery)
		}
		if previous == "blocked" && event.Status == "working" {
			unlock(Unstuck)
		}
		working := 0
		for _, status := range state.LastStatusByPane {
			if status == "working" {
				working++
			}
		}
		if working > state.PeakConcurrentWorking {
			state.PeakConcurrentWorking = working
		}
		if working >= 2 {
			unlock(DoubleTrouble)
		}
		if working >= 3 {
			unlock(FullHerd)
		}
	}
	sort.Strings(unlocked)
	return state, unlocked
}

func AchievementByID(id string) (Achievement, bool) {
	for _, achievement := range Catalog {
		if achievement.ID == id {
			return achievement, true
		}
	}
	return Achievement{}, false
}
