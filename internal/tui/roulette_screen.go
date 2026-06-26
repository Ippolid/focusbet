package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ippolid/focusbet/internal/domain"
	"github.com/Ippolid/focusbet/internal/game"
)

// rouletteBetOption is a selectable bet on the roulette screen.
type rouletteBetOption struct {
	bet   game.RouletteBet
	label string
}

var rouletteBets = []rouletteBetOption{
	{game.BetRed, "Red (×2)"},
	{game.BetBlack, "Black (×2)"},
	{game.BetEven, "Even (×2)"},
	{game.BetOdd, "Odd (×2)"},
	{game.BetNumber, "Number (×36)"},
}

// rouletteSpinFrames is how many animation frames the wheel cycles before the
// real pocket is revealed.
const rouletteSpinFrames = 16

// rouletteSpinMsg advances the roulette spin animation.
type rouletteSpinMsg struct{}

// rouletteScreen lets the player pick a bet + stake, spins, and settles.
type rouletteScreen struct {
	root    *model
	phase   playPhase
	cursor  int // index into rouletteBets
	number  int // chosen pocket for a straight bet
	stake   domain.Minutes
	pocket  int
	color   string
	out     game.Outcome
	err     error
	frame   int // animation frame counter while spinning
	display int // currently displayed pocket during the animation
	confettiHost
}

func newRoulette(root *model) *rouletteScreen {
	return &rouletteScreen{root: root, stake: defaultStake(root)}
}

func (r *rouletteScreen) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	km := r.root.keys

	if _, ok := msg.(rouletteSpinMsg); ok {
		if r.phase != playSpinning {
			return r, nil
		}
		r.frame++
		if r.frame >= rouletteSpinFrames {
			return r, r.settle() // draw the real pocket and settle, only now
		}
		// Cycle a shown pocket for the rolling effect.
		r.display = (r.display + 7) % 37
		return r, rouletteSpinTick(r.frame)
	}

	if _, ok := msg.(frameMsg); ok {
		return r, r.advanceConfetti()
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return r, nil
	}
	switch {
	case keyMatches(key, km.Quit):
		return r, r.root.quit()
	case keyMatches(key, km.Back):
		return r, switchTo(screenGames, nil)
	}
	if r.phase == playSpinning {
		return r, nil // ignore keys mid-spin
	}
	if r.phase == playSettled {
		// Enter places another bet; the player stays at the table.
		if keyMatches(key, km.Enter) {
			r.phase = playBetting
			r.err = nil
			r.frame = 0
			if bank := r.root.core.Bank().Bank(); r.stake > bank {
				r.stake = bank
			}
			if r.stake < stakeStep {
				r.stake = stakeStep
			}
		}
		return r, nil
	}

	switch {
	case keyMatches(key, km.Up):
		if r.cursor > 0 {
			r.cursor--
		}
	case keyMatches(key, km.Down):
		if r.cursor < len(rouletteBets)-1 {
			r.cursor++
		}
	case keyMatches(key, km.Left):
		if r.isNumberBet() {
			r.number = (r.number - 1 + 37) % 37
		} else {
			r.stake = adjustStake(r.root, r.stake, false)
		}
	case keyMatches(key, km.Right):
		if r.isNumberBet() {
			r.number = (r.number + 1) % 37
		} else {
			r.stake = adjustStake(r.root, r.stake, true)
		}
	case key.Type == tea.KeyShiftLeft:
		r.stake = adjustStake(r.root, r.stake, false)
	case key.Type == tea.KeyShiftRight:
		r.stake = adjustStake(r.root, r.stake, true)
	case keyMatches(key, km.Enter):
		return r, r.spin()
	}
	return r, nil
}

func (r *rouletteScreen) isNumberBet() bool {
	return rouletteBets[r.cursor].bet == game.BetNumber
}

// spin debits the stake and starts the wheel animation. The real pocket is not
// drawn until the animation finishes (see settle), so the bank balance shown
// during the spin never reveals the outcome early.
func (r *rouletteScreen) spin() tea.Cmd {
	if ok, reason := r.root.core.Bank().CanSpend(r.stake); !ok {
		r.err = fmt.Errorf("%s", reason)
		return nil
	}
	now := r.root.wallNow() // staking writes a money-log entry; use real time
	if err := r.root.core.Stake(now, domain.GameRoulette, r.stake); err != nil {
		r.phase = playSettled
		r.err = err
		return nil
	}
	// Fresh round: clear the previous result so a stale pocket/color can never be
	// shown if settle below errors out.
	r.out = game.Outcome{}
	r.pocket = 0
	r.color = ""
	r.err = nil
	r.phase = playSpinning
	r.frame = 0
	r.display = 0
	return rouletteSpinTick(0)
}

// settle draws the real pocket, credits the outcome, and moves to the settled
// phase. Called only when the animation ends, so the result is revealed exactly
// when the wheel visually stops.
func (r *rouletteScreen) settle() tea.Cmd {
	now := r.root.wallNow() // money-log entries use real time, not the scaled clock
	g := r.root.core.NewGame(domain.GameRoulette, game.NewCryptoRand())
	g.Start(r.stake)
	gs, _ := g.Step(game.Move{Bet: rouletteBets[r.cursor].bet, Number: r.number})
	r.out = g.Result()
	if len(gs.Symbols) == 2 {
		fmt.Sscanf(gs.Symbols[0], "%d", &r.pocket)
		r.color = gs.Symbols[1]
	}
	r.phase = playSettled
	if err := r.root.core.Settle(now, domain.GameRoulette, r.out); err != nil {
		r.err = err
	}
	if r.err == nil && r.out.Win {
		return r.celebrate(r.root.width) // confetti on a winning pocket
	}
	return nil
}

// rouletteSpinTick schedules the next animation frame, slowing toward the end.
func rouletteSpinTick(frame int) tea.Cmd {
	delay := time.Duration(40+frame*9) * time.Millisecond
	return tea.Tick(delay, func(time.Time) tea.Msg { return rouletteSpinMsg{} })
}

func (r *rouletteScreen) View() string {
	st := r.root.styles

	labels := make([]string, len(rouletteBets))
	for i, b := range rouletteBets {
		label := b.label
		if b.bet == game.BetNumber {
			label = fmt.Sprintf("Number %d (×36)", r.number)
		}
		labels[i] = label
	}

	lines := []string{
		st.subtitle.Render("Bank " + st.bank.Render(r.root.core.Bank().Bank().String()) + " rest"),
		"",
	}

	switch r.phase {
	case playSpinning:
		num := st.art.Render(bigText(fmt.Sprintf("%d", r.display), digitFont))
		lines = append(lines, num, "", st.subtitle.Render("spinning…"))
		body := lipgloss.JoinVertical(lipgloss.Center, lines...)
		return st.frame(r.root.width, r.root.height, st.heading.Render("ROULETTE"), body,
			"esc back • q quit")

	case playSettled:
		pocketStyle := st.good
		if !r.out.Win {
			pocketStyle = st.bad
		}
		lines = append(lines,
			pocketStyle.Render(bigText(fmt.Sprintf("%d", r.pocket), digitFont)),
			st.subtitle.Render(r.color),
			"",
		)
		if r.err != nil {
			lines = append(lines, st.bad.Render("error: "+r.err.Error()))
		} else if r.out.Win {
			lines = append(lines, st.good.Render(fmt.Sprintf("WIN ×%.0f  +%s",
				r.out.Multiplier, r.out.Payout.String())))
		} else {
			lines = append(lines, st.bad.Render("No win — stake lost"))
		}
		body := lipgloss.JoinVertical(lipgloss.Center, lines...)
		out := st.frame(r.root.width, r.root.height, st.heading.Render("ROULETTE"), body,
			"enter bet again • esc games • q quit")
		return r.renderConfetti(out, r.root.width, r.root.height)

	default:
		lines = append(lines,
			st.menu(labels, r.cursor),
			"",
			"Stake "+st.good.Render(r.stake.String()),
		)
		if r.err != nil {
			lines = append(lines, st.bad.Render(r.err.Error()))
		}
		body := lipgloss.JoinVertical(lipgloss.Center, lines...)
		footer := "↑/↓ bet • ←/→ stake (or number) • enter spin • esc back"
		return st.frame(r.root.width, r.root.height, st.heading.Render("ROULETTE"), body, footer)
	}
}
