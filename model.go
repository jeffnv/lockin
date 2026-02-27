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
type vizTickMsg struct{}
type togglePauseMsg struct{}
type blockerStoppedMsg struct{}

type model struct {
	totalDuration time.Duration
	remaining     time.Duration
	taskName      string
	blockApps     []string
	vizMode       string
	font *fontData

	paused bool
	done   bool

	width  int
	height int

	blockerStop   chan struct{}
	blockerPaused *atomic.Bool

	defragOriginal []uint8 // original random layout: 1=data, 0=free
	defragWidth    int

	sortFrames [][]int // pre-computed animation frames for sort vizs
	sortWidth  int     // elements per row

	lastTickAt time.Time // wall clock at last second-tick, for sub-second interpolation

	binaryPrevBits []bool      // previous bit states for phosphor fade
	binaryOnAt     []time.Time // when each bit last turned on
	binaryOffAt    []time.Time // when each bit last turned off

	barPrevFilled int       // previous filled count for slice animation
	barSliceAt    time.Time // when the current slice started animating

	dotPrevCells []bool      // previous on/off state for dot font phosphor
	dotOnAt      []time.Time // when each dot cell last turned on
}

func newModel(cfg config) model {
	m := model{
		totalDuration: cfg.duration,
		remaining:     cfg.duration,
		taskName:      cfg.taskName,
		blockApps:     cfg.blockApps,
		vizMode:       cfg.vizMode,
		font:          fonts[cfg.fontStyle],
		blockerStop:   make(chan struct{}),
		blockerPaused: &atomic.Bool{},
	}
	if m.isDotFont() {
		m.updateDotFade()
	}
	return m
}

func (m model) isDotFont() bool   { return m.font == fonts["dot"] }
func (m model) isBlockFont() bool { return m.font == fonts["block"] }

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func doVizTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return vizTickMsg{}
	})
}

func (m model) needsFastTick() bool {
	if m.isDotFont() {
		return true
	}
	switch m.vizMode {
	case "bar", "binary", "bubble", "merge", "quick":
		return true
	}
	return false
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{doTick()}
	if len(m.blockApps) > 0 {
		cmds = append(cmds, startBlocker(m.blockApps, m.blockerPaused, m.blockerStop))
	}
	if m.needsFastTick() {
		cmds = append(cmds, doVizTick())
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
		if m.vizMode == "bubble" || m.vizMode == "merge" || m.vizMode == "quick" {
			m.initSortGrid()
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
			if !m.paused && m.needsFastTick() {
				m.lastTickAt = time.Now()
				return m, doVizTick()
			}
			return m, nil
		}
		return m, nil

	case togglePauseMsg:
		m.paused = !m.paused
		m.blockerPaused.Store(m.paused)
		if !m.paused && m.needsFastTick() {
			m.lastTickAt = time.Now()
			return m, doVizTick()
		}
		return m, nil

	case vizTickMsg:
		if m.paused || m.done {
			return m, nil
		}
		if m.vizMode == "bar" {
			m.updateBarSlice()
		}
		return m, doVizTick()

	case tickMsg:
		if m.paused || m.done {
			return m, doTick()
		}
		m.remaining -= time.Second
		m.lastTickAt = time.Now()
		if m.vizMode == "binary" {
			m.updateBinaryFade()
		}
		if m.isDotFont() {
			m.updateDotFade()
		}
		// Lazy-init viz grids if WindowSizeMsg hasn't fired yet
		if m.vizMode == "defrag" && len(m.defragOriginal) == 0 && m.width > 0 {
			m.initDefragGrid()
		}
		if (m.vizMode == "bubble" || m.vizMode == "merge" || m.vizMode == "quick") && len(m.sortFrames) == 0 && m.width > 0 {
			m.initSortGrid()
		}
		if m.remaining <= 0 {
			m.remaining = 0
			m.done = true
			m.shutdown()
			return m, tea.Quit
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
			Foreground(lipgloss.Color("7"))
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
			Foreground(lipgloss.Color("11"))
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

const dotFlareDuration = 150 * time.Millisecond

func (m model) timerTimeStr() string {
	h := int(m.remaining.Hours())
	min := int(m.remaining.Minutes()) % 60
	sec := int(m.remaining.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d", h, min)
	}
	return fmt.Sprintf("%02d:%02d", min, sec)
}

type glyphCol struct {
	rows []string
}

func (m model) timerCols() []glyphCol {
	timeStr := m.timerTimeStr()
	var cols []glyphCol
	for i, ch := range timeStr {
		if rows, ok := m.font.digits[ch]; ok {
			cols = append(cols, glyphCol{rows})
		}
		if i < len(timeStr)-1 {
			nextCh := rune(timeStr[i+1])
			if ch != ':' && nextCh != ':' {
				spacerRows := make([]string, m.font.height)
				for j := range spacerRows {
					spacerRows[j] = "  "
				}
				cols = append(cols, glyphCol{spacerRows})
			}
		}
	}
	return cols
}

func (m model) buildTimerGrid() [][]rune {
	cols := m.timerCols()
	var grid [][]rune
	for row := 0; row < m.font.height; row++ {
		var rowRunes []rune
		for _, c := range cols {
			rowRunes = append(rowRunes, []rune(c.rows[row])...)
		}
		grid = append(grid, rowRunes)
	}
	return grid
}

func (m *model) updateDotFade() {
	grid := m.buildTimerGrid()
	if len(grid) == 0 {
		return
	}

	width := len(grid[0])
	total := m.font.height * width

	currentCells := make([]bool, total)
	for r, row := range grid {
		for c, ch := range row {
			if ch != ' ' {
				currentCells[r*width+c] = true
			}
		}
	}

	if len(m.dotPrevCells) != total {
		m.dotPrevCells = currentCells
		m.dotOnAt = make([]time.Time, total)
		now := time.Now()
		for i, on := range currentCells {
			if on {
				m.dotOnAt[i] = now
			}
		}
		return
	}

	now := time.Now()
	for i := range currentCells {
		if !m.dotPrevCells[i] && currentCells[i] {
			m.dotOnAt[i] = now
		}
	}
	m.dotPrevCells = currentCells
}

func (m model) renderBigTimer() string {
	baseColor := m.timerColor()
	cols := m.timerCols()

	isBlock := m.isBlockFont()
	isDot := m.isDotFont()

	if !isBlock && !isDot {
		// Default path: per-row gradient (slim font, etc.)
		var renderedRows []string
		for row := 0; row < m.font.height; row++ {
			color := shaderGradient(row, 0, m.font.height, len(cols), baseColor, 0)
			style := lipgloss.NewStyle().Foreground(color)
			var rowStr strings.Builder
			for _, c := range cols {
				rowStr.WriteString(style.Render(c.rows[row]))
			}
			renderedRows = append(renderedRows, rowStr.String())
		}
		return strings.Join(renderedRows, "\n")
	}

	// Build 2D rune grid for per-cell rendering
	var grid [][]rune
	for row := 0; row < m.font.height; row++ {
		var rowRunes []rune
		for _, c := range cols {
			rowRunes = append(rowRunes, []rune(c.rows[row])...)
		}
		grid = append(grid, rowRunes)
	}
	if len(grid) == 0 {
		return ""
	}
	gridWidth := len(grid[0])

	renderHeight := m.font.height
	if isBlock {
		renderHeight++ // extra row for shadow overhang
	}

	var shadowColor lipgloss.Color
	if isBlock {
		shadowColor = modifyColor(baseColor, func(c hsl) hsl {
			c.l = 0.18
			c.s = c.s * 0.4
			return c
		})
	}

	var renderedRows []string
	for row := 0; row < renderHeight; row++ {
		gradientColor := shaderGradient(row, 0, m.font.height, gridWidth, baseColor, 0)
		var rowStr strings.Builder

		for col := 0; col < gridWidth; col++ {
			var ch rune = ' '
			if row < m.font.height {
				ch = grid[row][col]
			}
			isFilled := ch != ' '

			if isBlock {
				hasShadow := false
				if row > 0 && col+1 < gridWidth {
					srcRow := row - 1
					if srcRow < m.font.height {
						hasShadow = grid[srcRow][col+1] != ' '
					}
				}
				if isFilled {
					rowStr.WriteString(lipgloss.NewStyle().Foreground(gradientColor).Render("█"))
				} else if hasShadow {
					rowStr.WriteString(lipgloss.NewStyle().Foreground(shadowColor).Render("█"))
				} else {
					rowStr.WriteRune(' ')
				}
			} else { // isDot
				cellIdx := row*gridWidth + col
				if isFilled {
					color := gradientColor
					if cellIdx < len(m.dotOnAt) && !m.dotOnAt[cellIdx].IsZero() {
						elapsed := time.Since(m.dotOnAt[cellIdx])
						if elapsed < dotFlareDuration {
							frac := float64(elapsed) / float64(dotFlareDuration)
							color = modifyColor(gradientColor, func(c hsl) hsl {
								c.l = clamp01(c.l + (1.0-frac)*0.4)
								return c
							})
						}
					}
					rowStr.WriteString(lipgloss.NewStyle().Foreground(color).Render("● "))
				} else {
					rowStr.WriteString("  ")
				}
			}
		}
		renderedRows = append(renderedRows, rowStr.String())
	}
	return strings.Join(renderedRows, "\n")
}

func (m model) renderBlockedApps() string {
	var parts []string
	for _, app := range m.blockApps {
		parts = append(parts, "🔒 "+app)
	}
	style := lipgloss.NewStyle().
		Foreground(colorDim)
	return style.Render(strings.Join(parts, "  "))
}
