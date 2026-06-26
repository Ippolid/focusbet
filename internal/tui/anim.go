package tui

import (
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// frameMsg drives sub-second visual animations (smooth bars, confetti, screen
// reveals). It is distinct from the once-per-second tickMsg so animations stay
// fluid without speeding up the logical clock.
type frameMsg struct{}

// frameInterval is the animation frame period — ~33 fps, smooth without
// flooding the terminal.
const frameInterval = 30 * time.Millisecond

// frameTick schedules the next animation frame.
func frameTick() tea.Cmd {
	return tea.Tick(frameInterval, func(time.Time) tea.Msg { return frameMsg{} })
}

// --- screen reveal -----------------------------------------------------------

// reveal animates a screen's entrance: the header types out letter by letter
// while the body fades in line by line. Screens embed it, tick it on frameMsg,
// and wrap their header/body through it in View. It is purely cosmetic — once
// done() is true the content renders in full.
type reveal struct {
	frame int
}

// revealFrames is how many frames the entrance lasts (~0.24s at frameInterval)
// — quick enough to read as a flourish without making the screen feel slow.
const revealFrames = 8

// step advances the reveal by one frame, returning whether more frames remain.
func (r *reveal) step() bool {
	if r.frame < revealFrames {
		r.frame++
	}
	return r.frame < revealFrames
}

// done reports whether the entrance animation has finished.
func (r *reveal) done() bool { return r.frame >= revealFrames }

// header reveals a heading with a typewriter effect: a growing prefix of the
// runes, then the whole thing once complete.
func (r *reveal) header(s string) string {
	runes := []rune(s)
	if r.done() || len(runes) == 0 {
		return s
	}
	// Map the first half of the reveal to 0..len so the header finishes typing
	// before the body has fully faded in.
	progress := float64(r.frame) / float64(revealFrames) * 2
	if progress > 1 {
		progress = 1
	}
	n := int(progress * float64(len(runes)))
	if n < 1 {
		n = 1
	}
	return string(runes[:n])
}

// body reveals a multi-line block one line at a time from the top, so the screen
// "assembles" downward. Lines not yet revealed render as blank to keep layout
// stable (no jump when they appear).
func (r *reveal) body(s string) string {
	if r.done() {
		return s
	}
	lines := strings.Split(s, "\n")
	// The body starts appearing in the second half of the reveal.
	progress := (float64(r.frame)/float64(revealFrames) - 0.5) * 2
	if progress < 0 {
		progress = 0
	}
	shown := int(progress * float64(len(lines)))
	for i := range lines {
		if i >= shown {
			lines[i] = "" // reserve the row, reveal it later
		}
	}
	return strings.Join(lines, "\n")
}

// easeToward moves shown a fraction of the way toward target, for a smooth bar
// that chases the real (once-per-second) progress without snapping. It returns
// the new shown value and whether it is still meaningfully short of target.
func easeToward(shown, target float64) (float64, bool) {
	const rate = 0.25 // portion of the remaining gap closed per frame
	gap := target - shown
	if gap < 0.0005 && gap > -0.0005 {
		return target, false
	}
	return shown + gap*rate, true
}

// --- smooth progress bar -----------------------------------------------------

// smoothBar renders a fixed-width bar filled to frac in [0,1] using fractional
// block characters, so the fill advances smoothly between whole cells instead of
// jumping a cell at a time. Width is the number of cells.
func smoothBar(frac float64, width int, fill, empty lipgloss.Style) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	// Eighth-block characters give 8 sub-steps per cell for a fluid edge.
	const eighths = " ▏▎▍▌▋▊▉█"
	total := frac * float64(width)
	full := int(total)
	rem := total - float64(full)

	var b strings.Builder
	b.WriteString(fill.Render(strings.Repeat("█", full)))
	if full < width {
		idx := int(rem * 8)
		if idx > 0 {
			b.WriteString(fill.Render(string([]rune(eighths)[idx])))
			full++
		}
	}
	if full < width {
		b.WriteString(empty.Render(strings.Repeat("░", width-full)))
	}
	return b.String()
}

// --- confetti ----------------------------------------------------------------

// confettiGlyphs are the particle shapes; confettiColors their lipgloss colors.
var (
	confettiGlyphs = []rune{'✦', '✧', '★', '*', '+', '·', '❄'}
	confettiColors = []lipgloss.Color{colorGold, colorFocus, colorBreak, colorText, lipgloss.Color("205")}
)

// confettiFrames is how many frames a celebration lasts (~1s at frameInterval).
const confettiFrames = 32

// confettiParticle is one falling glyph with a column, a start row, a fall speed
// (rows per frame), a glyph and a color.
type confettiParticle struct {
	col   int
	row   float64
	speed float64
	glyph rune
	style lipgloss.Style
}

// confetti is a short celebratory particle burst overlaid on a screen. It is
// seeded once for a given width and advanced one frame at a time.
type confetti struct {
	frame     int
	particles []confettiParticle
}

// newConfetti seeds a burst sized to the terminal width. It uses math/rand for
// a lively, non-repeating scatter (cosmetic only — no fairness concern).
func newConfetti(width int) *confetti {
	if width <= 0 {
		width = 60
	}
	n := width / 2 // density scales with width
	if n < 12 {
		n = 12
	}
	ps := make([]confettiParticle, n)
	for i := range ps {
		ps[i] = confettiParticle{
			col:   rand.Intn(width),
			row:   -float64(rand.Intn(6)), // stagger the start above the top edge
			speed: 0.6 + rand.Float64()*0.9,
			glyph: confettiGlyphs[rand.Intn(len(confettiGlyphs))],
			style: lipgloss.NewStyle().Foreground(confettiColors[rand.Intn(len(confettiColors))]),
		}
	}
	return &confetti{particles: ps}
}

// step advances the burst one frame; it returns whether the burst is still
// active (more frames to play).
func (c *confetti) step() bool {
	c.frame++
	for i := range c.particles {
		c.particles[i].row += c.particles[i].speed
	}
	return c.active()
}

// active reports whether the burst is still playing.
func (c *confetti) active() bool { return c.frame < confettiFrames }

// overlay paints the particles onto a w×h block of content, returning the
// composited frame. Particles outside the box are skipped. The underlying
// content is preserved everywhere a particle isn't drawn.
func (c *confetti) overlay(content string, w, h int) string {
	if w <= 0 || h <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	// Normalize to a grid of runes so we can paint at (row, col).
	grid := make([][]rune, h)
	for y := 0; y < h; y++ {
		row := make([]rune, w)
		for x := range row {
			row[x] = ' '
		}
		if y < len(lines) {
			for x, r := range []rune(lines[y]) {
				if x < w {
					row[x] = r
				}
			}
		}
		grid[y] = row
	}
	// Fade the particle set out near the end so it doesn't pop off.
	for _, p := range c.particles {
		y := int(p.row)
		if y < 0 || y >= h || p.col < 0 || p.col >= w {
			continue
		}
		// Only overwrite blank cells, so the result text stays readable.
		if grid[y][p.col] == ' ' {
			grid[y][p.col] = p.glyph
		}
	}
	var b strings.Builder
	for y, row := range grid {
		if y > 0 {
			b.WriteByte('\n')
		}
		// Re-style particle glyphs as we emit (cheap: glyphs are rare).
		writeConfettiLine(&b, row, c.particles)
	}
	return b.String()
}

// writeConfettiLine emits one grid row, coloring any cell that holds a particle
// glyph with that particle's style.
func writeConfettiLine(b *strings.Builder, row []rune, particles []confettiParticle) {
	for _, r := range row {
		styled := false
		for _, p := range particles {
			if r == p.glyph {
				b.WriteString(p.style.Render(string(r)))
				styled = true
				break
			}
		}
		if !styled {
			b.WriteRune(r)
		}
	}
}

// confettiHost is an embeddable helper that lets any screen play a win
// celebration with three small calls: celebrate to start it, advanceConfetti on
// each frameMsg, and renderConfetti to composite it over the final frame.
type confettiHost struct {
	c *confetti
}

// celebrate starts (or restarts) a confetti burst sized to width, returning the
// command that begins the frame loop.
func (h *confettiHost) celebrate(width int) tea.Cmd {
	h.c = newConfetti(width)
	return frameTick()
}

// advanceConfetti steps the burst on a frameMsg and returns the next frame
// command while it is still active, or nil once it finishes (and clears itself).
func (h *confettiHost) advanceConfetti() tea.Cmd {
	if h.c == nil {
		return nil
	}
	if h.c.step() {
		return frameTick()
	}
	h.c = nil
	return nil
}

// renderConfetti overlays the active burst on content; when no burst is playing
// it returns content unchanged.
func (h *confettiHost) renderConfetti(content string, w, h2 int) string {
	if h.c == nil {
		return content
	}
	return h.c.overlay(content, w, h2)
}
