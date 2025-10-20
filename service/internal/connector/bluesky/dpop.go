package bluesky

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func base64urlEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// calculateAccessTokenHash calculates the SHA-256 hash of the access token for the "ath" claim
func calculateAccessTokenHash(accessToken string) string {
	hash := sha256.Sum256([]byte(accessToken))
	return base64urlEncode(hash[:])
}

func createDPoPProof(privKey *ecdsa.PrivateKey, httpMethod, httpUrl, serverNonce, accessTokenHash string) (string, error) {
	// JWT header
	header := map[string]interface{}{
		"typ": "dpop+jwt",
		"alg": "ES256",
		"jwk": map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64urlEncode(privKey.PublicKey.X.Bytes()),
			"y":   base64urlEncode(privKey.PublicKey.Y.Bytes()),
		},
	}

	// JWT payload
	now := time.Now().Unix()
	payload := map[string]interface{}{
		"htu": httpUrl,
		"htm": httpMethod,
		"iat": now,
		"jti": uuid.NewString(),
	}

	// Include access token hash if provided
	if accessTokenHash != "" {
		payload["ath"] = accessTokenHash
	}

	// Only include nonce if provided (not empty)
	if serverNonce != "" {
		payload["nonce"] = serverNonce
	}

	// Encode parts
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)

	encodedHeader := base64urlEncode(headerJSON)
	encodedPayload := base64urlEncode(payloadJSON)

	dataToSign := encodedHeader + "." + encodedPayload

	// Sign
	hash := sha256.Sum256([]byte(dataToSign))
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		return "", err
	}

	// Concatenate r and s into a 64-byte signature
	sigBytes := make([]byte, 64)
	r.FillBytes(sigBytes[:32])
	s.FillBytes(sigBytes[32:])

	encodedSig := base64urlEncode(sigBytes)
	jwt := dataToSign + "." + encodedSig
	return jwt, nil
}

func generatePrivateKey() (*ecdsa.PrivateKey, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	return privKey, nil

}
