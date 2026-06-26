package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/game"
)

// stakeStep is the smallest stake increment: a quarter-minute (15 seconds), so
// stakes can be set with second-level granularity via fractional minutes.
const stakeStep = domain.Minutes(0.25)

// defaultStake returns the minimum stake, so the player dials up from the
// smallest wager rather than starting near their whole bank.
func defaultStake(root *model) domain.Minutes {
	return stakeStep
}

// adjustStake nudges a stake by ±stakeStep, clamped to [stakeStep, bank].
func adjustStake(root *model, stake domain.Minutes, up bool) domain.Minutes {
	if up {
		if stake+stakeStep <= root.core.Bank().Bank() {
			stake += stakeStep
		}
	} else if stake-stakeStep >= stakeStep {
		stake -= stakeStep
	}
	return stake
}

// playPhase is where a one-shot game screen is in its lifecycle.
type playPhase int

const (
	playBetting  playPhase = iota // choosing a stake
	playSpinning                  // animating
	playSettled                   // showing the result
)

// spinFrames is how many animation frames a slots spin lasts before settling.
const spinFrames = 14

// spinTickMsg advances the spin animation, distinct from the global tickMsg.
type spinTickMsg struct{}

func spinTick() tea.Cmd {
	return func() tea.Msg { return spinTickMsg{} }
}

// playScreen runs one slot machine: pick a stake, spin, settle.
type playScreen struct {
	root    *model
	phase   playPhase
	stake   domain.Minutes
	frame   int
	reels   []string
	out     game.Outcome
	err     error
	symbols []string
	confettiHost
}

func newPlay(root *model, _ playPayload) *playScreen {
	syms := game.SlotSymbols()
	return &playScreen{
		root:    root,
		stake:   defaultStake(root),
		reels:   []string{syms[0], syms[0], syms[0]},
		symbols: syms,
	}
}

func (p *playScreen) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := p.root.keys
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case keyMatches(msg, km.Quit):
			return p, p.root.quit()
		case keyMatches(msg, km.Back):
			return p, switchTo(screenGames, nil)
		}
		switch p.phase {
		case playBetting:
			return p, p.updateBetting(msg)
		case playSettled:
			// Enter spins again with the same stake; the player stays in the game.
			if keyMatches(msg, km.Enter) {
				p.playAgain()
			}
		}

	case spinTickMsg:
		if p.phase != playSpinning {
			return p, nil
		}
		p.frame++
		if p.frame >= spinFrames {
			return p, p.settle()
		}
		// Cycle visible symbols for the rolling effect; slow near the end (easing).
		for i := range p.reels {
			p.reels[i] = p.symbols[(p.frame*3+i*2)%len(p.symbols)]
		}
		return p, spinTickEased(p.frame)

	case frameMsg:
		return p, p.advanceConfetti()
	}
	return p, nil
}

// spinTickEased slows the spin as it approaches the end, for a satisfying stop.
func spinTickEased(frame int) tea.Cmd {
	delay := time.Duration(40+frame*8) * time.Millisecond // 40ms → ~150ms over the spin
	return tea.Tick(delay, func(time.Time) tea.Msg { return spinTickMsg{} })
}

func (p *playScreen) updateBetting(msg tea.KeyMsg) tea.Cmd {
	km := p.root.keys
	switch {
	case keyMatches(msg, km.Left):
		p.stake = adjustStake(p.root, p.stake, false)
	case keyMatches(msg, km.Right):
		p.stake = adjustStake(p.root, p.stake, true)
	case keyMatches(msg, km.Enter):
		if ok, reason := p.root.core.Bank().CanSpend(p.stake); !ok {
			p.err = fmt.Errorf("%s", reason)
			return nil
		}
		p.phase = playSpinning
		p.frame = 0
		return spinTick()
	}
	return nil
}

// playAgain resets the screen to the betting phase for another spin, keeping the
// last stake (clamped to the current bank).
func (p *playScreen) playAgain() {
	p.phase = playBetting
	p.err = nil
	p.frame = 0
	if bank := p.root.core.Bank().Bank(); p.stake > bank {
		p.stake = bank
	}
	if p.stake < stakeStep {
		p.stake = stakeStep
	}
}

// settle stakes, plays the real spin through the app core, and shows the outcome.
func (p *playScreen) settle() tea.Cmd {
	now := p.root.wallNow() // money-log entries use real time, not the scaled clock
	if err := p.root.core.Stake(now, domain.GameSlots, p.stake); err != nil {
		p.phase = playSettled
		p.err = err
		return nil
	}
	g := p.root.core.NewGame(domain.GameSlots, game.NewCryptoRand())
	g.Start(p.stake)
	gs, _ := g.Step(game.Move{})
	out := g.Result()
	p.out = out
	if len(gs.Symbols) == 3 {
		p.reels = gs.Symbols
	}
	p.phase = playSettled
	if err := p.root.core.Settle(now, domain.GameSlots, out); err != nil {
		p.err = err
	}
	if p.err == nil && out.Win {
		return p.celebrate(p.root.width) // rain confetti on a win
	}
	return nil
}

func (p *playScreen) View() string {
	st := p.root.styles

	reels := st.art.Render("[ " + strings.Join(p.reels, "  ") + " ]")
	lines := []string{
		st.subtitle.Render("Bank " + st.bank.Render(p.root.core.Bank().Bank().String()) + " rest"),
		"",
		reels,
		"",
	}
	footer := "esc back • q quit"

	switch p.phase {
	case playBetting:
		lines = append(lines, "Stake "+st.good.Render(p.stake.String()))
		if p.err != nil {
			lines = append(lines, st.bad.Render("can't spin: "+p.err.Error()))
		}
		footer = "←/→ stake • enter spin • esc back • q quit"

	case playSpinning:
		lines = append(lines, st.subtitle.Render("spinning…"))

	case playSettled:
		switch {
		case p.err != nil:
			lines = append(lines, st.bad.Render("error: "+p.err.Error()))
		case p.out.Win:
			lines = append(lines, st.good.Render(fmt.Sprintf("WIN ×%.2f  +%s",
				p.out.Multiplier, p.out.Payout.String())))
		default:
			lines = append(lines, st.bad.Render("No win — stake lost"))
		}
		footer = "enter spin again • esc games • q quit"
	}

	header := st.heading.Render("SLOTS")
	body := lipgloss.JoinVertical(lipgloss.Center, lines...)
	out := st.frame(p.root.width, p.root.height, header, body, footer)
	return p.renderConfetti(out, p.root.width, p.root.height)
}
