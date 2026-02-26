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
	case "binary":
		return m.renderBinary()
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
	emptyStyle := lipgloss.NewStyle().Foreground(colorDim)

	bar := filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))

	pct := fmt.Sprintf(" %d%%", int(frac*100))
	pctStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	return bar + pctStyle.Render(pct)
}

// --- Defrag ---

func (m *model) initDefragGrid() {
	w := m.defragGridWidth()
	h := 8
	total := w * h

	if m.defragWidth == w && len(m.defragOriginal) == total {
		return
	}

	m.defragWidth = w
	m.defragOriginal = make([]uint8, total)

	// Fill ~65% of cells with data, rest free
	dataCount := total * 65 / 100
	for i := 0; i < dataCount; i++ {
		m.defragOriginal[i] = 1
	}
	// Shuffle to create chaotic layout
	rand.Shuffle(total, func(i, j int) {
		m.defragOriginal[i], m.defragOriginal[j] = m.defragOriginal[j], m.defragOriginal[i]
	})
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

func (m model) renderDefrag() string {
	if len(m.defragOriginal) == 0 || m.defragWidth == 0 {
		return ""
	}

	total := len(m.defragOriginal)
	frac := m.progressFraction()
	cursor := int(frac * float64(total))
	if cursor > total {
		cursor = total
	}

	// Count data cells in the processed region (0..cursor)
	dataInProcessed := 0
	for i := 0; i < cursor; i++ {
		if m.defragOriginal[i] == 1 {
			dataInProcessed++
		}
	}

	dataStyle := lipgloss.NewStyle().Foreground(colorDefragData)
	fragStyle := lipgloss.NewStyle().Foreground(colorDefragFrag)
	freeStyle := lipgloss.NewStyle().Foreground(colorDefragFree)

	var rows []string
	for rowStart := 0; rowStart < total; rowStart += m.defragWidth {
		end := rowStart + m.defragWidth
		if end > total {
			end = total
		}
		var row strings.Builder
		for i := rowStart; i < end; i++ {
			if i < dataInProcessed {
				row.WriteString(dataStyle.Render("██"))
			} else if i < cursor {
				row.WriteString(freeStyle.Render("░░"))
			} else {
				if m.defragOriginal[i] == 1 {
					row.WriteString(fragStyle.Render("██"))
				} else {
					row.WriteString(freeStyle.Render("░░"))
				}
			}
		}
		rows = append(rows, row.String())
	}

	return strings.Join(rows, "\n")
}

// --- Binary (BCD) ---

func (m model) renderBinary() string {
	h := int(m.remaining.Hours())
	min := int(m.remaining.Minutes()) % 60
	sec := int(m.remaining.Seconds()) % 60

	type digitGroup struct {
		label  string
		digits []int
	}

	var groups []digitGroup
	if h > 0 {
		groups = append(groups, digitGroup{"H", []int{h / 10, h % 10}})
	}
	groups = append(groups, digitGroup{"M", []int{min / 10, min % 10}})
	groups = append(groups, digitGroup{"S", []int{sec / 10, sec % 10}})

	activeStyle := lipgloss.NewStyle().Foreground(m.timerColor())
	inactiveStyle := lipgloss.NewStyle().Foreground(colorDim)
	labelStyle := lipgloss.NewStyle().Foreground(colorDim)

	bitValues := []int{8, 4, 2, 1}
	var rows [4]strings.Builder
	var labelRow strings.Builder

	for gi, g := range groups {
		if gi > 0 {
			for r := 0; r < 4; r++ {
				rows[r].WriteString("  ")
			}
			labelRow.WriteString("  ")
		}
		for di, d := range g.digits {
			if di > 0 {
				for r := 0; r < 4; r++ {
					rows[r].WriteString(" ")
				}
				labelRow.WriteString(" ")
			}
			for r := 0; r < 4; r++ {
				if d&bitValues[r] != 0 {
					rows[r].WriteString(activeStyle.Render("██"))
				} else {
					rows[r].WriteString(inactiveStyle.Render("░░"))
				}
			}
			labelRow.WriteString(labelStyle.Render(g.label + " "))
		}
	}

	var result []string
	for _, r := range rows {
		result = append(result, r.String())
	}
	result = append(result, labelRow.String())

	return strings.Join(result, "\n")
}
