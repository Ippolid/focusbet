package tui

import (
	"os"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gen2brain/beeep"
)

// alert fires the unmissable end-of-phase signal across every channel at once:
// a desktop notification, a real system sound, and (as a fallback) the terminal
// bell. The caller pairs it with an in-TUI visual flash. It never blocks the UI.
//
// title/body populate the desktop notification (e.g. "Focus done", "Time for a
// 5 min break").
func alert(title, body string) tea.Cmd {
	return func() tea.Msg {
		// Desktop notification (+ beep on platforms that support it). beeep falls
		// back to the terminal bell when there's no notification daemon.
		_ = beeep.Alert(title, body, "")
		// A real chime on top, louder and clearer than the notification beep.
		if !playSystemSound() {
			_, _ = os.Stdout.WriteString("\a")
		}
		return nil
	}
}

// playSystemSound tries the platform's sound player on a known system sound,
// returning true if one was launched. It starts the process without waiting so
// the timer screen keeps rendering.
func playSystemSound() bool {
	type player struct {
		bin  string
		args []string
	}

	var candidates []player
	switch runtime.GOOS {
	case "darwin":
		// afplay ships with macOS; Glass/Ping are standard alert sounds.
		candidates = []player{
			{"afplay", []string{"/System/Library/Sounds/Glass.aiff"}},
			{"afplay", []string{"/System/Library/Sounds/Ping.aiff"}},
		}
	case "linux":
		candidates = []player{
			{"paplay", []string{"/usr/share/sounds/freedesktop/stereo/complete.oga"}},
			{"aplay", []string{"/usr/share/sounds/alsa/Front_Center.wav"}},
		}
	}

	for _, c := range candidates {
		bin, err := exec.LookPath(c.bin)
		if err != nil {
			continue
		}
		// Skip a candidate whose sound file doesn't exist, so we fall through to
		// the next option (or the bell) instead of launching a silent failure.
		if len(c.args) > 0 {
			if _, err := os.Stat(c.args[0]); err != nil {
				continue
			}
		}
		cmd := exec.Command(bin, c.args...)
		if err := cmd.Start(); err == nil {
			go func() { _ = cmd.Wait() }() // reap so it isn't a zombie
			return true
		}
	}
	return false
}
