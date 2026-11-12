package bluesky

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"math/big"
)

const p256ScalarBytes = 32

// DPoP keys are stored as the decimal form of the P-256 private scalar, the
// format written before PrivateKey.D was deprecated, so stored accounts keep
// working.
func encodeDPoPKey(priv *ecdsa.PrivateKey) (string, error) {
	raw, err := priv.Bytes()
	if err != nil {
		return "", fmt.Errorf("encode DPoP key: %w", err)
	}
	return new(big.Int).SetBytes(raw).String(), nil
}

func decodeDPoPKey(stored string) (*ecdsa.PrivateKey, error) {
	d, ok := new(big.Int).SetString(stored, 10)
	if !ok || d.Sign() <= 0 || d.BitLen() > p256ScalarBytes*8 {
		return nil, fmt.Errorf("invalid stored DPoP key")
	}
	return ecdsa.ParseRawPrivateKey(elliptic.P256(), d.FillBytes(make([]byte, p256ScalarBytes)))
}
