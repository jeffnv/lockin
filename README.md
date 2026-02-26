# lockin

A full-screen terminal focus timer. Set a duration, lock in, get it done.

## Install

```bash
go install github.com/jeffnv/lockin@latest
```

Or build from source:

```bash
git clone https://github.com/jeffnv/lockin.git
cd lockin
go build -o lockin .
```

## Usage

```bash
lockin <duration> [task name] [flags]
```

Duration uses Go's time format: `30s`, `5m`, `25m`, `1h`, `1h30m`.

### Examples

```bash
lockin 25m                                    # 25 minute timer
lockin 30m "deep work"                        # with a task label
lockin 25m --block Safari,Messages,Discord    # kill distracting apps
lockin 1h --viz defrag                        # with progress visualization
lockin 25m --font slim --viz binary           # slim font + BCD display
```

## Flags

| Flag | Options | Description |
|---|---|---|
| `--block` | `App1,App2,...` | Kill listed apps every 5s while the timer runs |
| `--viz` | `bar`, `defrag`, `binary` | Show a progress visualization below the timer |
| `--font` | `block`, `slim`, `dot` | Timer digit style (default: `block`) |

## Controls

| Key | Action |
|---|---|
| `space` | Pause / resume |
| `q` / `ctrl+c` | Quit |

Pause can also be toggled externally with `kill -USR1 <pid>`.

## Timer colors

The timer shifts color as time runs down:

- **Green** — more than 25% remaining
- **Yellow** — 10–25% remaining
- **Red** — under 10% remaining

## License

MIT
