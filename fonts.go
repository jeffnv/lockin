package main

import (
	"strings"
	"unicode/utf8"
)

type fontData struct {
	height int
	digits map[rune][]string
}

var fonts = map[string]*fontData{
	"block": &blockFont,
	"slim":  &slimFont,
	"dot":   &dotFont,
}

func init() {
	for _, f := range fonts {
		// Find max rune width across digit glyphs (skip ':')
		maxWidth := 0
		for ch, rows := range f.digits {
			if ch == ':' {
				continue
			}
			for _, row := range rows {
				if w := utf8.RuneCountInString(row); w > maxWidth {
					maxWidth = w
				}
			}
		}
		// Center-pad each digit row to maxWidth
		for ch, rows := range f.digits {
			if ch == ':' {
				continue
			}
			for i, row := range rows {
				w := utf8.RuneCountInString(row)
				if w < maxWidth {
					left := (maxWidth - w) / 2
					right := maxWidth - w - left
					rows[i] = strings.Repeat(" ", left) + row + strings.Repeat(" ", right)
				}
			}
		}
	}
}

func renderDigit(f *fontData, ch rune) string {
	rows, ok := f.digits[ch]
	if !ok {
		return string(ch)
	}
	return strings.Join(rows, "\n")
}

func fontSpacer(height int) string {
	rows := make([]string, height)
	for i := range rows {
		rows[i] = " "
	}
	return strings.Join(rows, "\n")
}

var blockFont = fontData{
	height: 7,
	digits: map[rune][]string{
		'0': {" ██████ ", "██    ██", "██    ██", "██    ██", "██    ██", "██    ██", " ██████ "},
		'1': {"    ██  ", "  ████  ", "    ██  ", "    ██  ", "    ██  ", "    ██  ", " ██████ "},
		'2': {" ██████ ", "██    ██", "      ██", "  ██████", "██      ", "██      ", "████████"},
		'3': {" ██████ ", "██    ██", "      ██", "  ██████", "      ██", "██    ██", " ██████ "},
		'4': {"██    ██", "██    ██", "██    ██", "████████", "      ██", "      ██", "      ██"},
		'5': {"████████", "██      ", "██      ", "██████  ", "      ██", "██    ██", " ██████ "},
		'6': {" ██████ ", "██      ", "██      ", "██████  ", "██    ██", "██    ██", " ██████ "},
		'7': {"████████", "      ██", "     ██ ", "    ██  ", "   ██   ", "  ██    ", "  ██    "},
		'8': {" ██████ ", "██    ██", "██    ██", " ██████ ", "██    ██", "██    ██", " ██████ "},
		'9': {" ██████ ", "██    ██", "██    ██", " ███████", "      ██", "      ██", " ██████ "},
		':': {"      ", "  ██  ", "  ██  ", "      ", "  ██  ", "  ██  ", "      "},
	},
}

var slimFont = fontData{
	height: 3,
	digits: map[rune][]string{
		'0': {"█▀▀█", "█  █", "█▄▄█"},
		'1': {"  █ ", "  █ ", " ███"},
		'2': {"▀▀▀█", "█▀▀▀", "█▄▄▄"},
		'3': {"▀▀▀█", " ▀▀█", "▄▄▄█"},
		'4': {"█  █", "▀▀▀█", "   █"},
		'5': {"█▀▀▀", "▀▀▀█", "▄▄▄█"},
		'6': {"█▀▀▀", "█▀▀█", "█▄▄█"},
		'7': {"▀▀▀█", "  █ ", " █  "},
		'8': {"█▀▀█", "█▀▀█", "█▄▄█"},
		'9': {"█▀▀█", "▀▀▀█", "▄▄▄█"},
		':': {" ▄▄ ", "    ", " ▀▀ "},
	},
}

var dotFont = fontData{
	height: 5,
	digits: map[rune][]string{
		'0': {" ●●● ", "●   ●", "●   ●", "●   ●", " ●●● "},
		'1': {"  ●  ", " ●●  ", "  ●  ", "  ●  ", " ●●● "},
		'2': {" ●●● ", "●   ●", "  ●● ", " ●   ", "●●●●●"},
		'3': {" ●●● ", "    ●", "  ●● ", "    ●", " ●●● "},
		'4': {"●   ●", "●   ●", "●●●●●", "    ●", "    ●"},
		'5': {"●●●●●", "●    ", "●●●● ", "    ●", "●●●● "},
		'6': {" ●●● ", "●    ", "●●●● ", "●   ●", " ●●● "},
		'7': {"●●●●●", "   ● ", "  ●  ", " ●   ", " ●   "},
		'8': {" ●●● ", "●   ●", " ●●● ", "●   ●", " ●●● "},
		'9': {" ●●● ", "●   ●", " ●●●●", "    ●", " ●●● "},
		':': {"     ", "  ●  ", "     ", "  ●  ", "     "},
	},
}
