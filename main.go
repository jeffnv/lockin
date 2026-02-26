package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type config struct {
	duration  time.Duration
	taskName  string
	blockApps []string
	vizMode   string
	fontStyle string
}

func parseArgs(args []string) config {
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	var cfg config
	var positional []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		case "--block":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --block requires an argument")
				os.Exit(1)
			}
			i++
			cfg.blockApps = strings.Split(args[i], ",")
		case "--viz":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --viz requires an argument")
				os.Exit(1)
			}
			i++
			switch args[i] {
			case "bar", "defrag", "binary", "bubble", "merge":
				cfg.vizMode = args[i]
			default:
				fmt.Fprintf(os.Stderr, "error: unknown viz mode %q (use bar, defrag, binary, bubble, or merge)\n", args[i])
				os.Exit(1)
			}
		case "--font":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --font requires an argument")
				os.Exit(1)
			}
			i++
			switch args[i] {
			case "block", "slim", "dot":
				cfg.fontStyle = args[i]
			default:
				fmt.Fprintf(os.Stderr, "error: unknown font %q (use block, slim, or dot)\n", args[i])
				os.Exit(1)
			}
		default:
			positional = append(positional, args[i])
		}
	}

	if len(positional) == 0 {
		printUsage()
		os.Exit(1)
	}

	d, err := time.ParseDuration(positional[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid duration %q: %v\n", positional[0], err)
		os.Exit(1)
	}
	if d <= 0 {
		fmt.Fprintln(os.Stderr, "error: duration must be positive")
		os.Exit(1)
	}
	cfg.duration = d

	if len(positional) > 1 {
		cfg.taskName = positional[1]
	}

	if cfg.fontStyle == "" {
		cfg.fontStyle = "block"
	}

	return cfg
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: lockin <duration> [task name] [flags]

Duration formats: 30s, 5m, 30m, 1h, 1h30m

Flags:
  --block App1,App2        Block apps while timer runs
  --viz bar|defrag|binary|bubble|merge
                           Visualization mode
  --font block|slim|dot    Timer font style

Examples:
  lockin 30m "deep work"
  lockin 25m --block Safari,Messages,Discord
  lockin 1h30m --viz defrag
  lockin 25m --font slim --viz binary`)
}

func listenSIGUSR1(p *tea.Program) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)
	for range sig {
		p.Send(togglePauseMsg{})
	}
}

func main() {
	cfg := parseArgs(os.Args[1:])
	m := newModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	go listenSIGUSR1(p)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fm, ok := finalModel.(model); ok && fm.remaining <= 0 {
		fmt.Printf("lockin: %s complete", cfg.duration)
		if cfg.taskName != "" {
			fmt.Printf(" — %s", cfg.taskName)
		}
		fmt.Println()
	}
}
