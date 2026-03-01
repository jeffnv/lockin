package main

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	yellowThreshold = 5 * time.Minute
	redThreshold    = 1 * time.Minute
)

var (
	colorGreen  = lipgloss.Color("#00FF00")
	colorYellow = lipgloss.Color("#FFFF00")
	colorRed    = lipgloss.Color("#FF0000")
)

func (m model) timerColor() lipgloss.Color {
	switch {
	case m.remaining <= redThreshold:
		return colorRed
	case m.remaining <= yellowThreshold:
		return colorYellow
	default:
		return colorGreen
	}
}
