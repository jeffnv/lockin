package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m model) progressFraction() float64 {
	if m.totalDuration == 0 {
		return 0
	}
	elapsed := m.totalDuration - m.remaining
	// Interpolate sub-second progress using wall clock
	if !m.paused && !m.done && !m.lastTickAt.IsZero() {
		elapsed += time.Since(m.lastTickAt)
	}
	// Finish viz with 10% of time remaining so the completed state is visible
	frac := float64(elapsed) / (float64(m.totalDuration) * 0.9)
	if frac > 1 {
		frac = 1
	}
	return frac
}

// neonPulse returns the current pulse position and width for a neon sweep effect.
func neonPulse(width float64) (pos, pulseWidth float64) {
	const period = 2.0 // seconds per sweep
	t := math.Mod(float64(time.Now().UnixMilli())/1000.0, period) / period
	pos = t * width
	pulseWidth = math.Max(width*0.05, 1.0)
	return
}

func pulseBoost(x, pulsePos, pulseWidth float64) float64 {
	dist := math.Abs(x - pulsePos)
	return math.Exp(-(dist * dist) / (2 * pulseWidth * pulseWidth))
}

func (m model) renderViz() string {
	switch m.vizMode {
	case "bar":
		return m.renderBar()
	case "defrag":
		return m.renderDefrag()
	case "binary":
		return m.renderBinary()
	case "bubble", "merge", "quick":
		return m.renderSort()
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

	baseColor := m.timerColor()
	pulsePos, pulseW := neonPulse(float64(maxWidth))

	var bar strings.Builder
	for i := 0; i < maxWidth; i++ {
		boost := pulseBoost(float64(i), pulsePos, pulseW)
		var base lipgloss.Color
		var ch string
		if i < filled {
			base = baseColor
			ch = "█"
		} else {
			base = colorDim
			ch = "░"
		}
		color := modifyColor(base, func(c hsl) hsl {
			c.l = clamp01(c.l + boost*0.35)
			return c
		})
		bar.WriteString(lipgloss.NewStyle().Foreground(color).Render(ch))
	}

	pct := fmt.Sprintf(" %d%%", int(frac*100))
	pctStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	return bar.String() + pctStyle.Render(pct)
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

// --- Sort (bubble / merge) ---

func rainbowColor(value, total int) lipgloss.Color {
	return rainbowColorL(value, total, 0.5)
}

func rainbowColorL(value, total int, lightness float64) lipgloss.Color {
	h := float64(value) / float64(total) * 0.85 // stop before wrapping back to red
	r, g, b := hslToRGB(hsl{h, 1.0, lightness})
	return lipgloss.Color(rgbToHex(r, g, b))
}

func (m *model) initSortGrid() {
	w := m.defragGridWidth()
	h := 4
	total := w * h

	if m.sortWidth == w && len(m.sortFrames) > 0 {
		return
	}

	m.sortWidth = w

	arr := make([]int, total)
	for i := range arr {
		arr[i] = i
	}
	rand.Shuffle(total, func(i, j int) {
		arr[i], arr[j] = arr[j], arr[i]
	})

	var frames [][]int
	switch m.vizMode {
	case "bubble":
		frames = bubbleSortFrames(arr)
	case "merge":
		frames = mergeSortFrames(arr)
	case "quick":
		frames = quickSortFrames(arr)
	}

	// Subsample to cap memory usage
	const maxFrames = 2000
	if len(frames) <= maxFrames {
		m.sortFrames = frames
		return
	}
	subsampled := make([][]int, maxFrames)
	for i := range subsampled {
		subsampled[i] = frames[i*(len(frames)-1)/(maxFrames-1)]
	}
	m.sortFrames = subsampled
}

func (m model) renderSort() string {
	if len(m.sortFrames) == 0 || m.sortWidth == 0 {
		return ""
	}

	frac := m.progressFraction()
	idx := int(frac * float64(len(m.sortFrames)-1))
	if idx >= len(m.sortFrames) {
		idx = len(m.sortFrames) - 1
	}

	frame := m.sortFrames[idx]
	total := len(frame)

	// Glow map: find most recent change per cell, decay over trail
	// No glow on the final frame — sort is done
	const trailLen = 4
	glow := make([]float64, total)
	if idx < len(m.sortFrames)-1 {
		for i := range frame {
			for j := idx; j > idx-trailLen && j > 0; j-- {
				if m.sortFrames[j][i] != m.sortFrames[j-1][i] {
					glow[i] = 1.0 - float64(idx-j)/float64(trailLen)
					break
				}
			}
		}
	}

	var rows []string
	for rowStart := 0; rowStart < total; rowStart += m.sortWidth {
		end := rowStart + m.sortWidth
		if end > total {
			end = total
		}
		var row strings.Builder
		for i := rowStart; i < end; i++ {
			// Snap glow to 3 discrete lightness levels (no modifyColor)
			var l float64
			switch {
			case glow[i] > 0.66:
				l = 0.7
			case glow[i] > 0.33:
				l = 0.5
			default:
				l = 0.3
			}
			color := rainbowColorL(frame[i], total, l)
			row.WriteString(lipgloss.NewStyle().Foreground(color).Render("██"))
		}
		rows = append(rows, row.String())
	}

	return strings.Join(rows, "\n")
}

func bubbleSortFrames(arr []int) [][]int {
	a := make([]int, len(arr))
	copy(a, arr)

	frames := [][]int{append([]int(nil), a...)}
	n := len(a)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-1-i; j++ {
			if a[j] > a[j+1] {
				a[j], a[j+1] = a[j+1], a[j]
				frames = append(frames, append([]int(nil), a...))
			}
		}
	}
	return frames
}

func mergeSortFrames(arr []int) [][]int {
	a := make([]int, len(arr))
	copy(a, arr)

	frames := [][]int{append([]int(nil), a...)}
	mergeSortRec(a, 0, len(a), &frames)
	return frames
}

func mergeSortRec(a []int, lo, hi int, frames *[][]int) {
	if hi-lo <= 1 {
		return
	}
	mid := (lo + hi) / 2
	mergeSortRec(a, lo, mid, frames)
	mergeSortRec(a, mid, hi, frames)
	mergeHalves(a, lo, mid, hi, frames)
}

func mergeHalves(a []int, lo, mid, hi int, frames *[][]int) {
	left := make([]int, mid-lo)
	right := make([]int, hi-mid)
	copy(left, a[lo:mid])
	copy(right, a[mid:hi])

	i, j, k := 0, 0, lo
	for i < len(left) && j < len(right) {
		if left[i] <= right[j] {
			a[k] = left[i]
			i++
		} else {
			a[k] = right[j]
			j++
		}
		k++
		*frames = append(*frames, append([]int(nil), a...))
	}
	for i < len(left) {
		a[k] = left[i]
		i++
		k++
		*frames = append(*frames, append([]int(nil), a...))
	}
	for j < len(right) {
		a[k] = right[j]
		j++
		k++
		*frames = append(*frames, append([]int(nil), a...))
	}
}

func quickSortFrames(arr []int) [][]int {
	a := make([]int, len(arr))
	copy(a, arr)

	frames := [][]int{append([]int(nil), a...)}
	quickSortRec(a, 0, len(a)-1, &frames)
	return frames
}

func quickSortRec(a []int, lo, hi int, frames *[][]int) {
	if lo >= hi {
		return
	}
	pivot := a[hi]
	i := lo
	for j := lo; j < hi; j++ {
		if a[j] < pivot {
			a[i], a[j] = a[j], a[i]
			*frames = append(*frames, append([]int(nil), a...))
			i++
		}
	}
	a[i], a[hi] = a[hi], a[i]
	*frames = append(*frames, append([]int(nil), a...))
	quickSortRec(a, lo, i-1, frames)
	quickSortRec(a, i+1, hi, frames)
}

// --- Binary (BCD) ---

const (
	binaryFlareDuration = 150 * time.Millisecond
	binaryFadeDuration  = 100 * time.Millisecond
)

func (m model) binaryDigits() []int {
	totalSec := int(m.remaining.Seconds())
	if totalSec < 0 {
		totalSec = 0
	}
	maxSec := int(m.totalDuration.Seconds())
	nDigits := 1
	for n := maxSec; n >= 10; n /= 10 {
		nDigits++
	}
	digits := make([]int, nDigits)
	for i := nDigits - 1; i >= 0; i-- {
		digits[i] = totalSec % 10
		totalSec /= 10
	}
	return digits
}

func (m *model) updateBinaryFade() {
	digits := m.binaryDigits()

	bitValues := []int{8, 4, 2, 1}
	totalBits := len(digits) * 4

	currentBits := make([]bool, totalBits)
	for di, d := range digits {
		for bi, bv := range bitValues {
			currentBits[di*4+bi] = d&bv != 0
		}
	}

	if len(m.binaryPrevBits) != totalBits {
		m.binaryPrevBits = currentBits
		m.binaryOnAt = make([]time.Time, totalBits)
		m.binaryOffAt = make([]time.Time, totalBits)
		now := time.Now()
		for i, on := range currentBits {
			if on {
				m.binaryOnAt[i] = now
			}
		}
		return
	}

	now := time.Now()
	for i := range currentBits {
		if !m.binaryPrevBits[i] && currentBits[i] {
			m.binaryOnAt[i] = now
		}
		if m.binaryPrevBits[i] && !currentBits[i] {
			m.binaryOffAt[i] = now
		}
	}
	m.binaryPrevBits = currentBits
}

func (m model) renderBinary() string {
	digits := m.binaryDigits()

	baseColor := m.timerColor()
	dimStyle := lipgloss.NewStyle().Foreground(colorDim)

	bitValues := []int{8, 4, 2, 1}
	var rows [4]strings.Builder

	for di, d := range digits {
		if di > 0 {
			for r := 0; r < 4; r++ {
				rows[r].WriteString("  ")
			}
		}
		for r := 0; r < 4; r++ {
			bitIdx := di*4 + r
			if d&bitValues[r] != 0 {
				color := baseColor
				if bitIdx < len(m.binaryOnAt) && !m.binaryOnAt[bitIdx].IsZero() {
					elapsed := time.Since(m.binaryOnAt[bitIdx])
					if elapsed < binaryFlareDuration {
						frac := float64(elapsed) / float64(binaryFlareDuration)
						color = modifyColor(baseColor, func(c hsl) hsl {
							c.l = clamp01(c.l + (1.0-frac)*0.4)
							return c
						})
					}
				}
				rows[r].WriteString(lipgloss.NewStyle().Foreground(color).Render("██"))
			} else {
				if bitIdx < len(m.binaryOffAt) && !m.binaryOffAt[bitIdx].IsZero() {
					elapsed := time.Since(m.binaryOffAt[bitIdx])
					if elapsed < binaryFadeDuration {
						fade := 1.0 - float64(elapsed)/float64(binaryFadeDuration)
						fadeColor := modifyColor(baseColor, func(c hsl) hsl {
							c.l = clamp01(fade * 0.03)
							c.s = c.s * fade * 0.2
							return c
						})
						rows[r].WriteString(lipgloss.NewStyle().Foreground(fadeColor).Render("░░"))
						continue
					}
				}
				rows[r].WriteString(dimStyle.Render("░░"))
			}
		}
	}

	var result []string
	for _, r := range rows {
		result = append(result, r.String())
	}

	return strings.Join(result, "\n")
}
