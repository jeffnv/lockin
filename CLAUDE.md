# lockin

Full-screen terminal focus timer built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Build & run

```bash
go build -o lockin .
./lockin 25m
```

## Project structure

Single `package main`, no subpackages.

| File | Purpose |
|---|---|
| `main.go` | CLI arg parsing, entry point |
| `model.go` | Bubble Tea model: Init/Update/View, timer logic, rendering |
| `blocker.go` | App blocker — kills named processes via `pkill` every 5s |
| `viz.go` | Progress visualizations: bar, defrag, binary (BCD), sort (bubble, merge, quick) |
| `fonts.go` | Big digit font definitions: block, slim, dot |
| `colors.go` | ANSI color constants, timer color thresholds |
| `shaders.go` | HSL color math, gradient shader for digit rendering |

## Architecture notes

- Bubble Tea drives the TUI loop; `model.Update` handles ticks, keys, and window resize
- Timer ticks once per second; color shifts green -> yellow -> red based on remaining fraction
- App blocking runs as a Bubble Tea command (goroutine), pausable via `atomic.Bool`
- `SIGUSR1` toggles pause externally (e.g., from a Raycast script)
- Defrag viz generates a random grid on init, then "defragments" it as progress advances
- Sort vizs (bubble, merge, quick) pre-compute animation frames from a shuffled rainbow array, subsampled to ~2000 frames; cursor glow highlights active changes
- Gradient shader converts ANSI indices to hex for HSL math, but rendered output uses terminal's ANSI palette
