package utils

import (
	"crypto/rand"
	"math/big"

	"github.com/alexedwards/argon2id"
	"runtime"
)

var defaultHashParams = argon2id.Params{
	Memory:      64 * 1024,
	Iterations:  4,
	Parallelism: uint8(runtime.NumCPU()),
	SaltLength:  16,
	KeyLength:   32,
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := argon2id.CreateHash(password, &defaultHashParams)

	return hashedPassword, err
}

func VerifyPassword(hashedPassword, password string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hashedPassword)

	return match, err
}

const generatedPasswordLength = 16
const generatedPasswordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GeneratePassword returns a cryptographically random password suitable for local accounts.
func GeneratePassword() (string, error) {
	password := make([]byte, generatedPasswordLength)
	max := big.NewInt(int64(len(generatedPasswordChars)))

	for i := range password {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		password[i] = generatedPasswordChars[n.Int64()]
	}

	return string(password), nil
}
