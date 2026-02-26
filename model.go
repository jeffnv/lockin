package main

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time
type togglePauseMsg struct{}
type blockerStoppedMsg struct{}

type model struct {
	totalDuration time.Duration
	remaining     time.Duration
	taskName      string
	blockApps     []string
	vizMode       string

	paused bool
	done   bool

	width  int
	height int

	blockerStop   chan struct{}
	blockerPaused *atomic.Bool

	defragCells []bool
	defragWidth int
}

func newModel(cfg config) model {
	return model{
		totalDuration: cfg.duration,
		remaining:     cfg.duration,
		taskName:      cfg.taskName,
		blockApps:     cfg.blockApps,
		vizMode:       cfg.vizMode,
		blockerStop:   make(chan struct{}),
		blockerPaused: &atomic.Bool{},
	}
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{doTick()}
	if len(m.blockApps) > 0 {
		cmds = append(cmds, startBlocker(m.blockApps, m.blockerPaused, m.blockerStop))
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.vizMode == "defrag" {
			m.initDefragGrid()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.done = true
			m.shutdown()
			return m, tea.Quit
		case " ":
			m.paused = !m.paused
			m.blockerPaused.Store(m.paused)
			return m, nil
		}
		return m, nil

	case togglePauseMsg:
		m.paused = !m.paused
		m.blockerPaused.Store(m.paused)
		return m, nil

	case tickMsg:
		if m.paused || m.done {
			return m, doTick()
		}
		m.remaining -= time.Second
		if m.remaining <= 0 {
			m.remaining = 0
			m.done = true
			m.shutdown()
			return m, tea.Quit
		}
		if m.vizMode == "defrag" {
			m.flipDefragCells()
		}
		return m, doTick()

	case blockerStoppedMsg:
		return m, nil
	}

	return m, nil
}

func (m *model) shutdown() {
	select {
	case <-m.blockerStop:
	default:
		close(m.blockerStop)
	}
}

func (m model) View() string {
	if m.done {
		return ""
	}

	var sections []string

	// Task name
	if m.taskName != "" {
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#AAAAAA"))
		sections = append(sections, style.Render(m.taskName))
	}

	// Spacer
	sections = append(sections, "")

	// Big timer digits
	sections = append(sections, m.renderBigTimer())

	// Pause indicator
	if m.paused {
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFAA00"))
		sections = append(sections, "")
		sections = append(sections, style.Render("PAUSED"))
	}

	// Visualization
	if m.vizMode != "" {
		sections = append(sections, "")
		sections = append(sections, m.renderViz())
	}

	// Blocked apps
	if len(m.blockApps) > 0 {
		sections = append(sections, "")
		sections = append(sections, m.renderBlockedApps())
	}

	body := lipgloss.JoinVertical(lipgloss.Center, sections...)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

func (m model) renderBigTimer() string {
	h := int(m.remaining.Hours())
	min := int(m.remaining.Minutes()) % 60
	sec := int(m.remaining.Seconds()) % 60

	var timeStr string
	if h > 0 {
		timeStr = fmt.Sprintf("%d:%02d:%02d", h, min, sec)
	} else {
		timeStr = fmt.Sprintf("%02d:%02d", min, sec)
	}

	color := m.timerColor()
	style := lipgloss.NewStyle().Foreground(color)

	var rendered []string
	for i, ch := range timeStr {
		rendered = append(rendered, style.Render(bigDigit(ch)))
		// Add spacer between digits (but not after colon or before colon)
		if i < len(timeStr)-1 {
			nextCh := rune(timeStr[i+1])
			if ch != ':' && nextCh != ':' {
				rendered = append(rendered, style.Render(digitSpacer()))
			}
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m model) renderBlockedApps() string {
	var parts []string
	for _, app := range m.blockApps {
		parts = append(parts, "🔒 "+app)
	}
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))
	return style.Render(strings.Join(parts, "  "))
}
