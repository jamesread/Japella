package bluesky

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDPoPKeyStoredRoundTrip(t *testing.T) {
	original, err := generatePrivateKey()
	require.NoError(t, err)

	stored, err := encodeStoredDPoPKey(original)
	require.NoError(t, err)

	// Stored keys are the private scalar in decimal, the format existing rows use.
	raw, err := original.Bytes()
	require.NoError(t, err)
	assert.Equal(t, new(big.Int).SetBytes(raw).String(), stored)

	loaded, err := decodeStoredDPoPKey(stored)
	require.NoError(t, err)
	assert.True(t, original.Equal(loaded))

	hash := sha256.Sum256([]byte("dpop"))
	sig, err := ecdsa.SignASN1(rand.Reader, loaded, hash[:])
	require.NoError(t, err)
	assert.True(t, ecdsa.VerifyASN1(&original.PublicKey, hash[:], sig))
}

func TestDecodeStoredDPoPKeyRejectsInvalid(t *testing.T) {
	for _, stored := range []string{"", "not-a-number", "0", "-5"} {
		_, err := decodeStoredDPoPKey(stored)
		assert.Error(t, err, "stored key %q", stored)
	}
}
