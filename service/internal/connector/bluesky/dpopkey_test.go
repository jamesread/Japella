package bluesky

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDPoPKeyRoundTrip(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	stored, err := encodeDPoPKey(priv)
	require.NoError(t, err)

	decoded, err := decodeDPoPKey(stored)
	require.NoError(t, err)
	assert.True(t, priv.Equal(decoded))
	assert.True(t, priv.PublicKey.Equal(&decoded.PublicKey))
}

func TestDPoPKeyStoredAsDecimalScalar(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	raw, err := priv.Bytes()
	require.NoError(t, err)

	stored, err := encodeDPoPKey(priv)
	require.NoError(t, err)
	assert.Equal(t, new(big.Int).SetBytes(raw).String(), stored)
}

func TestDecodeDPoPKeyLeadingZeroScalar(t *testing.T) {
	decoded, err := decodeDPoPKey("1")
	require.NoError(t, err)

	raw, err := decoded.Bytes()
	require.NoError(t, err)
	assert.Len(t, raw, p256ScalarBytes)
	assert.Equal(t, byte(1), raw[p256ScalarBytes-1])
}

func TestDecodeDPoPKeyRejectsInvalid(t *testing.T) {
	for _, stored := range []string{"", "abc", "0", "-5", new(big.Int).Lsh(big.NewInt(1), 300).String()} {
		_, err := decodeDPoPKey(stored)
		assert.Error(t, err, stored)
	}
}
