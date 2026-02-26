package main

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) progressFraction() float64 {
	if m.totalDuration == 0 {
		return 0
	}
	elapsed := m.totalDuration - m.remaining
	return float64(elapsed) / float64(m.totalDuration)
}

func (m model) renderViz() string {
	switch m.vizMode {
	case "bar":
		return m.renderBar()
	case "defrag":
		return m.renderDefrag()
	default:
		return ""
	}
}

// --- Bar ---

func (m model) renderBar() string {
	maxWidth := 60
	if m.width-4 < maxWidth {
		maxWidth = m.width - 4
	}
	if maxWidth < 10 {
		maxWidth = 10
	}

	frac := m.progressFraction()
	filled := int(frac * float64(maxWidth))
	if filled > maxWidth {
		filled = maxWidth
	}
	empty := maxWidth - filled

	color := m.timerColor()
	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))

	bar := filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))

	pct := fmt.Sprintf(" %d%%", int(frac*100))
	pctStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	return bar + pctStyle.Render(pct)
}

// --- Defrag ---

func (m *model) initDefragGrid() {
	w := m.defragGridWidth()
	h := 8
	total := w * h

	if m.defragWidth == w && len(m.defragCells) == total {
		return
	}

	m.defragWidth = w
	m.defragCells = make([]bool, total)

	// Pre-fill proportional to current progress
	target := int(m.progressFraction() * float64(total))
	perm := rand.Perm(total)
	for i := 0; i < target && i < len(perm); i++ {
		m.defragCells[perm[i]] = true
	}
}

func (m model) defragGridWidth() int {
	w := m.width * 6 / 10
	if w < 10 {
		w = 10
	}
	// Each cell renders as 2 chars (██ or ░░), so halve the available width
	w = w / 2
	return w
}

func (m *model) flipDefragCells() {
	if len(m.defragCells) == 0 {
		return
	}

	total := len(m.defragCells)
	target := int(m.progressFraction() * float64(total))

	filled := 0
	for _, c := range m.defragCells {
		if c {
			filled++
		}
	}

	// Flip random unfilled cells to reach target
	attempts := 0
	for filled < target && attempts < total*2 {
		idx := rand.Intn(total)
		if !m.defragCells[idx] {
			m.defragCells[idx] = true
			filled++
		}
		attempts++
	}
}

func (m model) renderDefrag() string {
	if len(m.defragCells) == 0 || m.defragWidth == 0 {
		return ""
	}

	color := m.timerColor()
	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#222222"))

	var rows []string
	for i := 0; i < len(m.defragCells); i += m.defragWidth {
		end := i + m.defragWidth
		if end > len(m.defragCells) {
			end = len(m.defragCells)
		}
		var row strings.Builder
		for _, c := range m.defragCells[i:end] {
			if c {
				row.WriteString(filledStyle.Render("██"))
			} else {
				row.WriteString(emptyStyle.Render("░░"))
			}
		}
		rows = append(rows, row.String())
	}

	return strings.Join(rows, "\n")
}
