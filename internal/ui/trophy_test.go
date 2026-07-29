package ui

import (
	"strings"
	"testing"

	"github.com/SerHappy/herdr-achievements/internal/achievements"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func settledState() achievements.State {
	state := achievements.NewState()
	for _, item := range achievements.Catalog {
		state.Unlocked[item.ID] = "2026-01-01T00:00:00Z"
		state.Seen[item.ID] = true
	}
	return state
}

func update(t *testing.T, model Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := model.Update(msg)
	result, ok := next.(Model)
	if !ok {
		t.Fatalf("model type = %T", next)
	}
	return result
}

func TestFormatUnlockedAt(t *testing.T) {
	for _, test := range []struct {
		timestamp string
		want      string
	}{
		{timestamp: "2026-07-28T20:58:28.525335Z", want: "Jul 28 · 20:58 UTC"},
		{timestamp: "2026-07-28T22:58:28+02:00", want: "Jul 28 · 20:58 UTC"},
		{timestamp: "not-a-timestamp", want: "not-a-timestamp"},
	} {
		if got := formatUnlockedAt(test.timestamp); got != test.want {
			t.Errorf("formatUnlockedAt(%q) = %q, want %q", test.timestamp, got, test.want)
		}
	}
}

func TestNavigationStaysInsideCatalog(t *testing.T) {
	model := NewModel(settledState())
	model = update(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if model.SelectedID() != achievements.FirstHoof {
		t.Fatalf("k from first selected %q", model.SelectedID())
	}
	for range len(achievements.Catalog) + 2 {
		model = update(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	if model.SelectedID() != achievements.FullHerd {
		t.Fatalf("j from last selected %q", model.SelectedID())
	}
}

func TestRevealCanBeSkippedWithoutChangingReplayState(t *testing.T) {
	state := achievements.NewState()
	state.Unlocked[achievements.FirstHoof] = "2026-01-01T00:00:00Z"
	model := NewModel(state)
	if !model.IsRevealing() {
		t.Fatal("new achievement did not start a reveal")
	}
	model = update(t, model, revealTick{sequence: model.sequence})
	if !model.IsRevealing() || model.revealReady {
		t.Fatal("animation tick closed or completed the reveal too early")
	}
	model = update(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !model.IsRevealing() || !model.revealReady {
		t.Fatal("key did not skip directly to the settled reveal")
	}
	if got := model.SeenIDs(); len(got) != 0 {
		t.Fatalf("seen before confirming reveal = %v", got)
	}
	model = update(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.IsRevealing() {
		t.Fatal("enter did not confirm the settled reveal")
	}
	if got := model.SeenIDs(); len(got) != 1 || got[0] != achievements.FirstHoof {
		t.Fatalf("seen after confirm = %v", got)
	}

	model = NewModel(settledState())
	model = update(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if !model.IsRevealing() {
		t.Fatal("enter did not replay selected reveal")
	}
	model = update(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !model.IsRevealing() || !model.revealReady {
		t.Fatal("replay skip did not settle the reveal")
	}
	model = update(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if got := model.SeenIDs(); len(got) != 0 {
		t.Fatalf("replay changed seen state: %v", got)
	}
}

func TestRevealShowsProgressThroughNewTrophies(t *testing.T) {
	state := achievements.NewState()
	state.Unlocked[achievements.FirstHoof] = "2026-01-01T00:00:00Z"
	state.Unlocked[achievements.FirstDelivery] = "2026-01-01T00:00:00Z"
	model := NewModel(state)
	if !strings.Contains(model.View(), "NEW 1/2") {
		t.Fatalf("first reveal progress missing: %q", model.View())
	}
	model = update(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = update(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if !model.IsRevealing() || !strings.Contains(model.View(), "NEW 2/2") {
		t.Fatalf("second reveal progress missing: %q", model.View())
	}
}

func TestNewBadgeFitsInside72ColumnList(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	state := achievements.NewState()
	for _, item := range achievements.Catalog {
		state.Unlocked[item.ID] = "2026-01-01T00:00:00Z"
	}
	model := NewModel(state)
	for _, line := range strings.Split(model.renderList(27), "\n") {
		if strings.TrimSpace(line) == "NEW" {
			t.Fatalf("NEW badge wrapped onto its own list row: %q", model.renderList(27))
		}
	}
}

func TestLockedAchievementCannotReplay(t *testing.T) {
	model := NewModel(achievements.NewState())
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{' '}},
	} {
		model = update(t, model, key)
		if model.IsRevealing() {
			t.Fatal("locked achievement started a replay")
		}
		if got := model.SeenIDs(); len(got) != 0 {
			t.Fatalf("locked replay changed seen state: %v", got)
		}
	}
}

func TestWrapTextKeepsUnlockConditionsInsideDetailCard(t *testing.T) {
	for _, text := range []string{
		"To unlock: Move an agent from blocked back to working.",
		"To unlock: Keep two agents working at the same time.",
		"To unlock: Keep three agents working at the same time.",
	} {
		lines := wrapText(text, 24)
		if got := strings.Join(lines, " "); got != text {
			t.Fatalf("wrapped text = %q, want %q", got, text)
		}
		for _, line := range lines {
			if width := lipgloss.Width(line); width > 24 {
				t.Fatalf("wrapped line width = %d, want <= 24: %q", width, line)
			}
		}
	}
}

func TestLockedFullHerdTextDoesNotWrapAtTheCardEdge(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	model := NewModel(achievements.NewState())
	model.selected = len(achievements.Catalog) - 1
	// At 72 columns the right card receives 41 columns after the list and gap.
	detail := model.renderDetail(41)
	for _, want := range []string{
		"Three agents were working at once.",
		"at the same time.",
		"Progress: 0 / 3 agents working",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("locked FULL HERD text was split: missing %q in %q", want, detail)
		}
	}
	for _, line := range wrapText("To unlock: Keep three agents working at the same time.", cardContentWidth(41)) {
		if width := lipgloss.Width(line); width >= cardContentWidth(41)+1 {
			t.Fatalf("detail row leaves no terminal safety margin: %d %q", width, line)
		}
	}
}

func TestRenderingAtPopupSizesAndWithoutColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, size := range []tea.WindowSizeMsg{{Width: 50, Height: 14}, {Width: 72, Height: 20}, {Width: 100, Height: 30}} {
		model := update(t, NewModel(settledState()), size)
		assertFitsViewport(t, model.View(), size)
		if !strings.Contains(model.View(), "HERDR TROPHY ROOM") || !strings.Contains(model.View(), "FIRST HOOF") {
			t.Fatalf("%dx%d room is not readable: %q", size.Width, size.Height, model.View())
		}
		if size.Width == 72 && size.Height == 20 && strings.HasPrefix(strings.TrimLeft(model.View(), " "), "╭") {
			t.Fatal("72x20 room still has an outer Trophy Room border")
		}
		lines := strings.Split(model.View(), "\n")
		if !strings.Contains(lines[len(lines)-1], "↑↓/j/k select") {
			t.Fatalf("%dx%d footer is not pinned to the bottom: %q", size.Width, size.Height, lines[len(lines)-1])
		}
		model = update(t, model, tea.KeyMsg{Type: tea.KeyEnter})
		view := model.View()
		assertFitsViewport(t, view, size)
		if !strings.Contains(view, "TROPHY UNLOCKED") && size.Width >= 25 && size.Height >= 13 {
			t.Fatalf("%dx%d reveal is missing its title: %q", size.Width, size.Height, view)
		}
		if strings.Contains(view, "HERDR TROPHY ROOM") {
			t.Fatalf("%dx%d reveal appended to the room instead of taking it over", size.Width, size.Height)
		}
		if strings.Contains(view, "\x1b[") {
			t.Fatalf("%dx%d view emitted color escapes with NO_COLOR", size.Width, size.Height)
		}
	}
}

func TestTinyRoomFitsViewport(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	size := tea.WindowSizeMsg{Width: 10, Height: 7}
	model := update(t, NewModel(settledState()), size)

	assertFitsViewport(t, model.View(), size)
	if !strings.HasPrefix(model.View(), "HERDR TROP") {
		t.Fatalf("tiny room header was not truncated as expected: %q", model.View())
	}
}

func TestTinyRevealShowsTheAcceptedContinuationKeys(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	size := tea.WindowSizeMsg{Width: 24, Height: 12}
	model := update(t, NewModel(settledState()), size)
	model = update(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if view := model.View(); !strings.Contains(view, "Any key skips") {
		t.Fatalf("tiny reveal skip prompt missing: %q", view)
	}

	model = update(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if view := model.View(); !strings.Contains(view, "Enter/Space continue") {
		t.Fatalf("tiny reveal continuation prompt missing: %q", view)
	}

	model = update(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !model.IsRevealing() {
		t.Fatal("tiny reveal accepted an unsupported continuation key")
	}
	model = update(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.IsRevealing() {
		t.Fatal("tiny reveal did not accept Enter")
	}
}

func TestArtworkHasOneFixedDisplayCanvas(t *testing.T) {
	for id, badge := range artwork {
		if len(badge.lines) != artHeight {
			t.Fatalf("%s has %d lines, want %d", id, len(badge.lines), artHeight)
		}
		for row, line := range badge.lines {
			if width := lipgloss.Width(line); width != artWidth {
				t.Fatalf("%s row %d display width = %d, want %d", id, row, width, artWidth)
			}
			if width := lipgloss.Width(line); width > artWidth {
				t.Fatalf("%s row %d exceeds canvas: %d", id, row, width)
			}
		}
	}
}

func assertFitsViewport(t *testing.T, view string, size tea.WindowSizeMsg) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) > size.Height {
		t.Fatalf("%dx%d view has %d lines", size.Width, size.Height, len(lines))
	}
	for index, line := range lines {
		if width := lipgloss.Width(line); width > size.Width {
			t.Fatalf("%dx%d row %d has width %d", size.Width, size.Height, index, width)
		}
	}
}
