package game

import (
	"math"
	"testing"

	"github.com/Ippolid/focusbet/internal/economy"
)

// seqRand is a deterministic Rand for tests: Intn cycles a fixed script of
// reel indices, so we can drive slots to a known outcome. Float64 is unused here.
type seqRand struct {
	seq []int
	i   int
}

func (r *seqRand) Intn(n int) int {
	// The slots reel maps a weighted draw in [0,totalWeight) to a symbol. To land
	// on a chosen symbol index we return the cumulative-weight start of it.
	v := r.seq[r.i%len(r.seq)]
	r.i++
	return v % n
}
func (r *seqRand) Float64() float64 { return 0 }

// weightStart returns the Intn value that selects symbol index idx on the strip.
func weightStart(idx int) int {
	start := 0
	for i := 0; i < idx; i++ {
		start += slotStrip[i].weight
	}
	return start
}

func TestSlots_ThreeOfAKind(t *testing.T) {
	// Force three "7️⃣" (index 5): triple payout 160×.
	seven := weightStart(5)
	s := NewSlots(&seqRand{seq: []int{seven, seven, seven}})

	s.Start(10)
	state, done := s.Step(Move{})
	if !done || !state.Done {
		t.Fatal("spin should be done")
	}
	out := s.Result()
	if out.Multiplier != 160 {
		t.Errorf("multiplier = %v, want 160", out.Multiplier)
	}
	if out.Payout != 1600 { // 10 × 160
		t.Errorf("payout = %v, want 1600", out.Payout)
	}
	if !out.Win {
		t.Error("Win = false, want true")
	}
	if out.Fraction != 1.0 { // 160 == ceiling
		t.Errorf("fraction = %v, want 1.0", out.Fraction)
	}
}

func TestSlots_Pair(t *testing.T) {
	cherry := weightStart(0)
	lemon := weightStart(1)
	s := NewSlots(&seqRand{seq: []int{cherry, cherry, lemon}})

	s.Start(10)
	s.Step(Move{})
	out := s.Result()
	if out.Multiplier != pairMult {
		t.Errorf("multiplier = %v, want %v", out.Multiplier, pairMult)
	}
	if out.Win { // 0.8× < stake, so not a win
		t.Error("pair should not be a Win")
	}
}

func TestSlots_NoMatch(t *testing.T) {
	s := NewSlots(&seqRand{seq: []int{
		weightStart(0), weightStart(1), weightStart(2),
	}})
	s.Start(10)
	s.Step(Move{})
	out := s.Result()
	if out.Multiplier != 0 || out.Payout != 0 || out.Win {
		t.Errorf("no-match outcome = %+v, want zero loss", out)
	}
	if out.Fraction != 0 {
		t.Errorf("fraction = %v, want 0", out.Fraction)
	}
}

// TestSlots_RTP enumerates the full independent-reel distribution and asserts
// the analytic RTP sits in the intended corridor.
func TestSlots_RTP(t *testing.T) {
	n := len(slotStrip)
	total := float64(slotTotalWeight)
	prob := func(i int) float64 { return float64(slotStrip[i].weight) / total }

	var ev float64
	for a := 0; a < n; a++ {
		for b := 0; b < n; b++ {
			for c := 0; c < n; c++ {
				p := prob(a) * prob(b) * prob(c)
				counts := map[int]int{a: 0, b: 0, c: 0}
				counts[a]++
				counts[b]++
				counts[c]++
				best, bestSym := 0, -1
				for sym, cnt := range counts {
					if cnt > best {
						best, bestSym = cnt, sym
					}
				}
				var mult float64
				switch best {
				case 3:
					mult = slotStrip[bestSym].tripleMult
				case 2:
					mult = pairMult
				}
				ev += p * mult
			}
		}
	}

	if !economy.WithinRTPCorridor(ev, 0.88, 0.92) {
		t.Errorf("slots RTP = %.4f, want in [0.88, 0.92]", ev)
	}
}

func TestProvablyFair_Deterministic(t *testing.T) {
	seed := []byte("server-secret")
	a := NewProvablyFair(seed, "client", 1)
	b := NewProvablyFair(seed, "client", 1)

	for i := 0; i < 100; i++ {
		if a.Intn(37) != b.Intn(37) {
			t.Fatalf("provably-fair diverged at draw %d", i)
		}
	}
	// Different nonce -> different stream (extremely likely).
	c := NewProvablyFair(seed, "client", 2)
	same := true
	a2 := NewProvablyFair(seed, "client", 1)
	for i := 0; i < 20; i++ {
		if a2.Intn(1000) != c.Intn(1000) {
			same = false
			break
		}
	}
	if same {
		t.Error("nonce 1 and 2 produced identical streams")
	}
}

func TestProvablyFair_Commitment(t *testing.T) {
	seed := []byte("abc")
	if got := Commitment(seed); len(got) != 64 {
		t.Errorf("commitment len = %d, want 64 hex chars", len(got))
	}
}

func TestCryptoRand_Uniformish(t *testing.T) {
	r := NewCryptoRand()
	const buckets, draws = 6, 60000
	hist := make([]int, buckets)
	for i := 0; i < draws; i++ {
		hist[r.Intn(buckets)]++
	}
	expected := float64(draws) / buckets
	for i, h := range hist {
		if math.Abs(float64(h)-expected)/expected > 0.1 { // within 10%
			t.Errorf("bucket %d count %d deviates >10%% from %.0f", i, h, expected)
		}
	}
}
