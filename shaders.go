package main

import (
	"fmt"
	"math"

	"github.com/charmbracelet/lipgloss"
)

type shaderFunc func(row, col, height, cols int, base lipgloss.Color, t float64) lipgloss.Color

// --- Color math ---

type hsl struct {
	h, s, l float64
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func hexToRGB(hex string) (uint8, uint8, uint8) {
	var r, g, b uint8
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

func rgbToHex(r, g, b uint8) string {
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func rgbToHSL(r, g, b uint8) hsl {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))

	l := (max + min) / 2.0

	if max == min {
		return hsl{0, 0, l}
	}

	d := max - min
	var s float64
	if l > 0.5 {
		s = d / (2.0 - max - min)
	} else {
		s = d / (max + min)
	}

	var h float64
	switch max {
	case rf:
		h = (gf - bf) / d
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/d + 2
	case bf:
		h = (rf-gf)/d + 4
	}
	h /= 6

	return hsl{h, s, l}
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func hslToRGB(c hsl) (uint8, uint8, uint8) {
	if c.s == 0 {
		v := uint8(clamp01(c.l) * 255)
		return v, v, v
	}

	var q float64
	if c.l < 0.5 {
		q = c.l * (1 + c.s)
	} else {
		q = c.l + c.s - c.l*c.s
	}
	p := 2*c.l - q

	r := uint8(clamp01(hueToRGB(p, q, c.h+1.0/3.0)) * 255)
	g := uint8(clamp01(hueToRGB(p, q, c.h)) * 255)
	b := uint8(clamp01(hueToRGB(p, q, c.h-1.0/3.0)) * 255)

	return r, g, b
}

func modifyColor(base lipgloss.Color, fn func(hsl) hsl) lipgloss.Color {
	hex := string(base)
	if len(hex) > 0 && hex[0] != '#' {
		if h, ok := ansiToHex[hex]; ok {
			hex = h
		}
	}
	r, g, b := hexToRGB(hex)
	c := rgbToHSL(r, g, b)
	c = fn(c)
	c.l = clamp01(c.l)
	c.s = clamp01(c.s)
	nr, ng, nb := hslToRGB(c)
	return lipgloss.Color(rgbToHex(nr, ng, nb))
}

// shaderGradient applies a top-lit vertical gradient, respects timer color.
func shaderGradient(row, col, height, cols int, base lipgloss.Color, t float64) lipgloss.Color {
	if height <= 1 {
		return base
	}
	frac := float64(row) / float64(height-1) // 0 at top, 1 at bottom
	return modifyColor(base, func(c hsl) hsl {
		c.l = clamp01(c.l * (1.15 - frac*0.65))
		return c
	})
}
