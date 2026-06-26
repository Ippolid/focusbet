// Package game holds the games and the RNG that drives them. Games are pure
// with respect to the bank: they take a stake, draw from a Rand, and report an
// Outcome (winnings + game_fraction). They never touch the store or the bank —
// the caller settles the Outcome through the balance core.
package game

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Rand is the randomness source games depend on. It yields uniform values, so a
// game expresses its odds purely through how it consumes them.
type Rand interface {
	// Intn returns a uniform int in [0, n). For n <= 0 it returns 0 rather than
	// panicking, so a caller bug degrades gracefully instead of crashing the app.
	Intn(n int) int
	// Float64 returns a uniform float in [0, 1).
	Float64() float64
}

// CryptoRand is the default Rand, backed by crypto/rand. It is unpredictable but
// not reproducible — use ProvablyFair when verifiability is required.
type CryptoRand struct{}

// NewCryptoRand returns a crypto/rand-backed Rand.
func NewCryptoRand() CryptoRand { return CryptoRand{} }

// Intn implements Rand using rejection sampling to avoid modulo bias.
func (CryptoRand) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return intn(n, readUint64Crypto)
}

// Float64 implements Rand.
func (CryptoRand) Float64() float64 {
	return float64(readUint64Crypto()>>11) / (1 << 53)
}

func readUint64Crypto() uint64 {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// crypto/rand failing is catastrophic and unrecoverable for a game of chance.
		panic(fmt.Sprintf("game: crypto/rand failed: %v", err))
	}
	return binary.BigEndian.Uint64(buf[:])
}

// ProvablyFair is a commit-reveal Rand. The server commits to serverSeed by
// publishing its hash before play; each draw is
// HMAC-SHA256(serverSeed, clientSeed:nonce:counter), so once serverSeed is
// revealed the player can recompute every outcome and verify fairness.
//
// nonce identifies the game (one per play); counter advances per draw within it.
type ProvablyFair struct {
	serverSeed []byte
	clientSeed string
	nonce      uint64
	counter    uint64
}

// NewProvablyFair builds a provably-fair Rand for one game. serverSeed is the
// revealed-later secret, clientSeed is the player's chosen string, and nonce is
// the per-game counter from state.
func NewProvablyFair(serverSeed []byte, clientSeed string, nonce uint64) *ProvablyFair {
	return &ProvablyFair{serverSeed: serverSeed, clientSeed: clientSeed, nonce: nonce}
}

// Commitment returns hex(SHA256(serverSeed)), shown to the player before play.
func Commitment(serverSeed []byte) string {
	sum := sha256.Sum256(serverSeed)
	return hex.EncodeToString(sum[:])
}

// block returns the next 8 random bytes as a uint64 and advances the counter.
func (p *ProvablyFair) block() uint64 {
	msg := fmt.Sprintf("%s:%d:%d", p.clientSeed, p.nonce, p.counter)
	p.counter++
	mac := hmac.New(sha256.New, p.serverSeed)
	mac.Write([]byte(msg))
	sum := mac.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}

// Intn implements Rand.
func (p *ProvablyFair) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return intn(n, p.block)
}

// Float64 implements Rand.
func (p *ProvablyFair) Float64() float64 {
	return float64(p.block()>>11) / (1 << 53)
}

// intn maps a uint64 source to a uniform [0,n) via rejection sampling, shared by
// both Rand implementations so they have identical, unbiased distributions.
func intn(n int, next func() uint64) int {
	un := uint64(n)
	limit := (^uint64(0)/un)*un - 1 // largest multiple of n that fits, minus one
	for {
		v := next()
		if v <= limit {
			return int(v % un)
		}
	}
}
