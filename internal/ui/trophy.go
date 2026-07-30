// Package ui contains the presentation-only Trophy Room terminal interface.
package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/SerHappy/herdr-achievements/internal/achievements"
)

const (
	revealFrameDuration = 110 * time.Millisecond
	revealFrames        = artHeight + 2
	artWidth            = 21
	artHeight           = 7
)

type art struct {
	condition string
	lines     []string
}

var artwork = makeArtwork()

func makeArtwork() map[string]art {
	return map[string]art{
		achievements.FirstHoof: {
			condition: "Let the first agent join your herd.",
			lines: normalizeArtwork([]string{
				"     ░░░░░░░░",
				"  ░░░░░░░░░░░░▄██▄▀",
				" ░░░░░░░░░░░░█████",
				" ░░░░░░░░░░░██░███",
				"  ░░░░░░░░░░░▀███▀",
				"   ██     ██",
				"   ▀▀     ▀▀",
			}),
		},
		achievements.FirstDelivery: {
			condition: "Finish a stretch of agent work.",
			lines: normalizeArtwork([]string{
				"    ▄▄▄▄▄▄▄▄▄▄▄▄▄▄",
				"    ████   ███   █",
				"    █   ███   ████",
				"    █▄▄▄▀▀▀▄▄▄▀▀▀█",
				"    ████▄▄▄███▄▄▄█",
				"    █",
				"  ▄▄█▄▄▄",
			}),
		},
		achievements.Unstuck: {
			condition: "Move an agent from blocked back to working.",
			lines: normalizeArtwork([]string{
				"        ▄▀▀▀▄",
				"      ▄▀         ▄",
				"      █         ▀▄▀",
				"   █▀▀▀▀▀▀▀▀▀▀▀▀▀█",
				"   █     ▄█▄     █",
				"   █      █      █",
				"   ▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀",
			}),
		},
		achievements.DoubleTrouble: {
			condition: "Keep two agents working at the same time.",
			lines: normalizeArtwork([]string{
				"░░░░             ░░░░",
				"░░░░░▄██▄▀ ▀▄██▄░░░░░",
				"░░░░█████   █████░░░░",
				"░░░██░███   ███░██░░░",
				"░░░░▀███▀   ▀███▀░░░░",
				"██ ██           ██ ██",
				"▀▀ ▀▀           ▀▀ ▀▀",
			}),
		},
		achievements.FullHerd: {
			condition: "Keep three agents working at the same time.",
			lines: normalizeArtwork([]string{
				"  ▄███████████████▄",
				"▄█▀▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▀█▄",
				"█▒▒▒▒░░░▒░░░▒░░░▒▒▒▒█",
				"█▒▒▒▒▀█▀▒▀█▀▒▀█▀▒▒▒▒█",
				"█▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒█",
				"▀█▄▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▄█▀",
				"  ▀███████████████▀",
			}),
		},
	}
}

// normalizeArtwork preserves the horizontal coordinates chosen by the artist.
// It only pads lines on the right to form a fixed 21×7 canvas.
func normalizeArtwork(raw []string) []string {
	if len(raw) > artHeight {
		panic(fmt.Sprintf("artwork has %d rows, maximum is %d", len(raw), artHeight))
	}
	lines := make([]string, artHeight)
	for i := range lines {
		line := ""
		if i < len(raw) {
			line = raw[i]
		}
		width := lipgloss.Width(line)
		if width > artWidth {
			panic(fmt.Sprintf("artwork row %d has width %d, maximum is %d", i, width, artWidth))
		}
		lines[i] = line + strings.Repeat(" ", artWidth-width)
	}
	return lines
}

func truncateLine(line string, width int) string {
	var b strings.Builder
	for _, r := range line {
		candidate := b.String() + string(r)
		if lipgloss.Width(candidate) > width {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func formatUnlockedAt(timestamp string) string {
	unlockedAt, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return timestamp
	}
	return unlockedAt.UTC().Format("Jan 2 · 15:04 UTC")
}

var (
	gold     = lipgloss.Color("#F6C453")
	green    = lipgloss.Color("#7BAE5A")
	cream    = lipgloss.Color("#F5E9C8")
	muted    = lipgloss.Color("#86807A")
	amber    = lipgloss.Color("#E6A64C")
	title    = lipgloss.NewStyle().Bold(true).Foreground(cream)
	subtle   = lipgloss.NewStyle().Foreground(muted)
	selected = lipgloss.NewStyle().Bold(true).Foreground(gold)
	unlocked = lipgloss.NewStyle().Foreground(green)
	locked   = lipgloss.NewStyle().Foreground(muted)
	newBadge = lipgloss.NewStyle().Bold(true).Foreground(amber)
	keyHint  = lipgloss.NewStyle().Foreground(muted)
	confetti = lipgloss.NewStyle().Foreground(gold)
	border   = lipgloss.RoundedBorder()
)

type revealTick struct{ sequence int }

// Model is independent from persistence. SeenIDs reports only reveals which
// completed or were skipped during this program run.
type Model struct {
	state       achievements.State
	selected    int
	width       int
	height      int
	pending     []string
	revealing   string
	revealNew   bool
	revealStep  int
	revealReady bool
	revealTotal int
	revealIndex int
	sequence    int
	sessionSeen map[string]bool
}

func NewModel(state achievements.State) Model {
	m := Model{
		state:       state,
		width:       88,
		height:      26,
		pending:     achievements.UnseenUnlocked(state),
		sessionSeen: map[string]bool{},
	}
	m.revealTotal = len(m.pending)
	m.startNextReveal()
	return m
}

func (m Model) Init() tea.Cmd {
	if m.revealing == "" {
		return nil
	}
	return m.revealTimer()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(1, msg.Width)
		m.height = max(1, msg.Height)
		return m, nil
	case revealTick:
		if m.revealing == "" || msg.sequence != m.sequence {
			return m, nil
		}
		if m.revealStep < revealFrames {
			m.revealStep++
		}
		if m.revealStep >= revealFrames {
			m.revealReady = true
			return m, nil
		}
		return m, m.revealTimer()
	case tea.KeyPressMsg:
		key := msg.String()
		runeKey := keyRune(msg)
		if m.revealing != "" {
			if key == "q" || key == "esc" || key == "ctrl+c" {
				m.finishReveal()
				return m, tea.Quit
			}
			if !m.revealReady {
				m.completeRevealAnimation()
				return m, nil
			}
			if key == "enter" || key == "space" {
				m.finishReveal()
				return m, m.startNextReveal()
			}
			return m, nil
		}
		switch key {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up":
			m.moveSelection(-1)
		case "down":
			m.moveSelection(1)
		case "enter", "space":
			if m.startReplay() {
				return m, m.revealTimer()
			}
		}
		switch runeKey {
		case 'k':
			m.moveSelection(-1)
		case 'j':
			m.moveSelection(1)
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	var content string
	if m.revealing != "" {
		content = m.fitHeight(m.renderRevealTakeover())
	} else {
		content = m.fitHeight(m.renderRoom())
	}
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (m Model) SelectedID() string {
	if len(achievements.Catalog) == 0 {
		return ""
	}
	return achievements.Catalog[m.selected].ID
}

func (m Model) SeenIDs() []string {
	ids := make([]string, 0, len(m.sessionSeen))
	for _, item := range achievements.Catalog {
		if m.sessionSeen[item.ID] {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func (m Model) IsRevealing() bool { return m.revealing != "" }

func (m *Model) moveSelection(delta int) {
	if len(achievements.Catalog) == 0 {
		return
	}
	m.selected = max(0, min(len(achievements.Catalog)-1, m.selected+delta))
}

func (m *Model) startNextReveal() tea.Cmd {
	if len(m.pending) == 0 {
		return nil
	}
	m.revealing = m.pending[0]
	m.pending = m.pending[1:]
	m.revealNew = true
	m.revealIndex++
	m.revealStep = 0
	m.revealReady = false
	m.sequence++
	return m.revealTimer()
}

func (m *Model) startReplay() bool {
	id := m.SelectedID()
	if _, unlocked := m.state.Unlocked[id]; !unlocked {
		return false
	}
	m.revealing = id
	m.revealNew = false
	m.revealStep = 0
	m.revealReady = false
	m.sequence++
	return true
}

func (m Model) revealTimer() tea.Cmd {
	sequence := m.sequence
	return tea.Tick(revealFrameDuration, func(time.Time) tea.Msg { return revealTick{sequence: sequence} })
}

func (m *Model) finishReveal() {
	if m.revealing != "" && m.revealNew {
		m.sessionSeen[m.revealing] = true
	}
	m.revealing = ""
	m.revealNew = false
	m.revealStep = 0
	m.revealReady = false
}

func (m *Model) completeRevealAnimation() {
	m.revealStep = revealFrames
	m.revealReady = true
}

func keyRune(msg tea.KeyPressMsg) rune {
	runes := []rune(msg.Key().Text)
	if len(runes) == 1 {
		return unicode.ToLower(runes[0])
	}
	return 0
}

func (m Model) renderRoom() string {
	if m.width < 25 || m.height < 8 {
		return m.renderTinyRoom()
	}
	innerWidth := max(1, m.width-2)
	header := m.renderHeader()
	var body string
	if m.isCompact() {
		body = m.renderCompact(innerWidth)
	} else {
		leftWidth := max(27, innerWidth*36/100)
		rightWidth := max(24, innerWidth-leftWidth-2)
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.renderList(leftWidth), "  ", m.renderDetail(rightWidth))
	}
	room := m.pinFooter(header+"\n\n"+body, m.renderHelp())
	return lipgloss.NewStyle().Width(m.width).Padding(0, 1).Render(room)
}

func (m Model) renderTinyRoom() string {
	lines := []string{"HERDR TROPHY ROOM", achievements.Catalog[m.selected].Name}
	for i := range lines {
		lines[i] = truncateLine(lines[i], m.width)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderHeader() string {
	count := 0
	for _, item := range achievements.Catalog {
		if _, ok := m.state.Unlocked[item.ID]; ok {
			count++
		}
	}
	barWidth := max(8, min(26, m.width-46))
	filled := 0
	if len(achievements.Catalog) > 0 {
		filled = count * barWidth / len(achievements.Catalog)
	}
	percent := 0
	if len(achievements.Catalog) > 0 {
		percent = count * 100 / len(achievements.Catalog)
	}
	bar := unlocked.Render(strings.Repeat("█", filled)) + locked.Render(strings.Repeat("░", barWidth-filled))
	return title.Render("HERDR TROPHY ROOM") + "  " + subtle.Render(fmt.Sprintf("%d / %d", count, len(achievements.Catalog))) + "  " + bar + "  " + subtle.Render(fmt.Sprintf("%d%%", percent))
}

func (m Model) renderList(width int) string {
	rows := make([]string, 0, len(achievements.Catalog)+1)
	rows = append(rows, title.Render("TROPHIES"))
	for index, item := range achievements.Catalog {
		_, isUnlocked := m.state.Unlocked[item.ID]
		marker, style := "◇", locked
		if isUnlocked {
			marker, style = "✓", unlocked
		}
		label := marker + " " + item.Name
		if isUnlocked && !m.hasSeen(item.ID) {
			label += " " + newBadge.Render("NEW")
		}
		if index == m.selected {
			rows = append(rows, selected.Render("› "+label))
		} else {
			rows = append(rows, "  "+style.Render(label))
		}
	}
	return card(width, green, strings.Join(rows, "\n"))
}

func (m Model) renderDetail(width int) string {
	if len(achievements.Catalog) == 0 {
		return ""
	}
	item := achievements.Catalog[m.selected]
	meta := artwork[item.ID]
	_, isUnlocked := m.state.Unlocked[item.ID]
	status := locked.Render("LOCKED")
	if isUnlocked {
		status = unlocked.Render("UNLOCKED")
		if !m.hasSeen(item.ID) {
			status += " " + newBadge.Render("NEW")
		}
	}
	contentWidth := cardContentWidth(width)
	lines := []string{title.Render(item.Name), status, "", m.renderArt(item.ID, isUnlocked, contentWidth), "", renderSubtleWrapped(item.Description, contentWidth)}
	if isUnlocked {
		lines = append(lines, subtle.Render("Unlocked "+formatUnlockedAt(m.state.Unlocked[item.ID])))
	} else {
		lines = append(lines, renderSubtleWrapped("To unlock: "+meta.condition, contentWidth), subtle.Render(m.progress(item)))
	}
	return card(width, m.detailBorder(isUnlocked), strings.Join(lines, "\n"))
}

func (m Model) renderCompact(width int) string {
	item := achievements.Catalog[m.selected]
	_, isUnlocked := m.state.Unlocked[item.ID]
	status := locked.Render("LOCKED")
	if isUnlocked {
		status = unlocked.Render("UNLOCKED")
	}
	contentWidth := cardContentWidth(width)
	lines := []string{subtle.Render(fmt.Sprintf("%d / %d  %s", m.selected+1, len(achievements.Catalog), status)), title.Render(item.Name), m.renderArt(item.ID, isUnlocked, contentWidth)}
	if isUnlocked {
		lines = append(lines, subtle.Render("Unlocked "+formatUnlockedAt(m.state.Unlocked[item.ID])))
	} else {
		lines = append(lines, renderSubtleWrapped("To unlock: "+artwork[item.ID].condition, contentWidth), subtle.Render(m.progress(item)))
	}
	return card(width, m.detailBorder(isUnlocked), strings.Join(lines, "\n"))
}

// renderRevealTakeover replaces the room instead of appending a modal below
// it. The normal form has a 13-row card, so it fits even in a 50x14 popup.
func (m Model) renderRevealTakeover() string {
	item, ok := achievements.AchievementByID(m.revealing)
	if !ok {
		return ""
	}
	if m.width < 25 || m.height < 13 {
		return m.renderTinyReveal(item)
	}
	cardWidth := min(35, m.width)
	revealContentWidth := max(1, cardWidth-8)
	visibleRows := max(0, min(artHeight, m.revealStep-2))
	borderColor := gold
	prompt := "any key skips"
	if m.revealReady {
		prompt = "Enter/Space continue"
	}
	lines := []string{
		m.renderConfetti(revealContentWidth),
		m.renderRevealTitle(),
		selected.Render(item.Name),
		m.renderArtRows(item.ID, true, max(artWidth, revealContentWidth), visibleRows),
	}
	panel := card(cardWidth, borderColor, strings.Join(lines, "\n"))
	status := "REPLAY"
	if m.revealNew {
		status = fmt.Sprintf("NEW %d/%d", m.revealIndex, m.revealTotal)
	}
	message := keyHint.Render(status + " • " + prompt)
	content := panel + "\n" + lipgloss.NewStyle().Width(cardWidth).Align(lipgloss.Center).Render(message)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) renderTinyReveal(item achievements.Achievement) string {
	prompt := "Any key skips"
	if m.revealReady {
		prompt = "Enter/Space continue"
	}
	lines := []string{"TROPHY", item.Name, prompt}
	for i := range lines {
		lines[i] = truncateLine(lines[i], m.width)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderArt(id string, isUnlocked bool, width int) string {
	return m.renderArtRows(id, isUnlocked, width, artHeight)
}

func (m Model) renderArtRows(id string, isUnlocked bool, width, visibleRows int) string {
	meta := artwork[id]
	style := locked
	if isUnlocked {
		style = lipgloss.NewStyle().Foreground(gold)
	}
	lines := make([]string, artHeight)
	for i := range lines {
		if i < visibleRows {
			lines[i] = meta.lines[i]
		} else {
			lines[i] = strings.Repeat(" ", artWidth)
		}
	}
	canvas := style.Render(strings.Join(lines, "\n"))
	if width < artWidth {
		return canvas
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(canvas)
}

func card(width int, borderColor color.Color, content string) string {
	if width < 5 {
		return content
	}
	return lipgloss.NewStyle().Width(width).Border(border).BorderForeground(borderColor).Padding(0, 1).Render(content)
}

// Leave one column empty inside a card. Some terminals eagerly wrap a styled
// line that lands exactly on the final content column, losing its color reset.
func cardContentWidth(width int) int { return max(1, width-5) }

func (m Model) renderHelp() string {
	return keyHint.Render("↑↓/j/k select  •  Enter replay unlocked  •  Esc close")
}

func (m Model) renderConfetti(width int) string {
	frames := []string{
		"",
		"·     ·",
		"·  ✦  ·  ◆  ·",
		"·  ✦  ·  ◆  ·  ✦  ·",
	}
	index := min(len(frames)-1, m.revealStep)
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(confetti.Render(frames[index]))
}

func (m Model) renderRevealTitle() string {
	return title.Render("TROPHY UNLOCKED")
}

func renderSubtleWrapped(text string, width int) string {
	lines := wrapText(text, width)
	for i, line := range lines {
		lines[i] = subtle.Render(line)
	}
	return strings.Join(lines, "\n")
}

// wrapText wraps before ANSI styling is applied. This keeps every terminal
// row inside the detail card and gives each row its own style reset.
func wrapText(text string, width int) []string {
	if width < 1 {
		return nil
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, len(words))
	line := ""
	for _, word := range words {
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if lipgloss.Width(candidate) <= width {
			line = candidate
			continue
		}
		if line != "" {
			lines = append(lines, line)
			line = ""
		}
		for lipgloss.Width(word) > width {
			part := truncateLine(word, width)
			lines = append(lines, part)
			word = strings.TrimPrefix(word, part)
		}
		line = word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func (m Model) progress(item achievements.Achievement) string {
	if item.Target == 0 {
		return "Progress: waiting for this moment"
	}
	progress := min(m.state.PeakConcurrentWorking, item.Target)
	return fmt.Sprintf("Progress: %d / %d agents working", progress, item.Target)
}

func (m Model) detailBorder(isUnlocked bool) color.Color {
	if isUnlocked {
		return gold
	}
	return muted
}

func (m Model) hasSeen(id string) bool { return m.state.Seen[id] || m.sessionSeen[id] }

func (m Model) isCompact() bool { return m.width < 72 || m.height < 20 }

func (m Model) pinFooter(content, footer string) string {
	if m.height < 2 {
		return footer
	}
	contentLines := strings.Split(content, "\n")
	maxContentLines := m.height - 1
	if len(contentLines) > maxContentLines {
		contentLines = contentLines[:maxContentLines]
	}
	content = strings.Join(contentLines, "\n")
	spacerLines := max(0, m.height-lipgloss.Height(content)-1)
	return content + strings.Repeat("\n", spacerLines+1) + footer
}

// fitHeight ensures a stale or very small WindowSizeMsg never makes Bubble
// Tea scroll the alternate screen. All normal popup widths already fit.
func (m Model) fitHeight(view string) string {
	lines := strings.Split(view, "\n")
	if len(lines) <= m.height {
		return view
	}
	return strings.Join(lines[:max(1, m.height)], "\n")
}
