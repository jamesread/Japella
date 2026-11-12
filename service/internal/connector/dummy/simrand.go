package dummy

import "math/rand/v2"

// simIntN picks simulated content and engagement for the dummy connector;
// nothing it produces is security-sensitive, so a fast PRNG is appropriate.
func simIntN(n int) int {
	return rand.IntN(n) //nolint:gosec // simulated data only
}
