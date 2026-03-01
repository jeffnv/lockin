package main

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen  = lipgloss.Color("10") // bright green
	colorYellow = lipgloss.Color("11") // bright yellow
	colorRed    = lipgloss.Color("9")  // bright red

	colorDim = lipgloss.Color("8") // dark gray

	colorDefragData = lipgloss.Color("12") // bright blue
	colorDefragFrag = lipgloss.Color("9")  // bright red
	colorDefragFree = lipgloss.Color("0")  // black
)

// ansiToHex maps ANSI color indices to standard RGB hex values.
// Used by the gradient shader for HSL math since ANSI indices
// don't carry RGB info. The rendered output still uses the user's
// terminal-configured ANSI colors.
var ansiToHex = map[string]string{
	"0": "#000000", "1": "#aa0000", "2": "#00aa00", "3": "#aa5500",
	"4": "#0000aa", "5": "#aa00aa", "6": "#00aaaa", "7": "#aaaaaa",
	"8": "#555555", "9": "#ff5555", "10": "#55ff55", "11": "#ffff55",
	"12": "#5555ff", "13": "#ff55ff", "14": "#55ffff", "15": "#ffffff",
}

func (m model) timerColor() lipgloss.Color {
	frac := float64(m.remaining) / float64(m.totalDuration)
	switch {
	case frac <= 0.10:
		return colorRed
	case frac <= 0.25:
		return colorYellow
	default:
		return colorGreen
	}
}
