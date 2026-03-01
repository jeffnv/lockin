package main

import (
	"os/exec"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func startBlocker(apps []string, paused *atomic.Bool, stop chan struct{}) tea.Cmd {
	return func() tea.Msg {
		killApps(apps)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return blockerStoppedMsg{}
			case <-ticker.C:
				if !paused.Load() {
					killApps(apps)
				}
			}
		}
	}
}

func killApps(apps []string) {
	for _, app := range apps {
		_ = exec.Command("pkill", "-x", app).Run()
	}
}
