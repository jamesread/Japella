package utils

import (
	"runtime"
	"github.com/alexedwards/argon2id"
)

var defaultHashParams = argon2id.Params {
	Memory: 64 * 1024,
	Iterations: 4,
	Parallelism: uint8(runtime.NumCPU()),
	SaltLength: 16,
	KeyLength: 32,
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := argon2id.CreateHash(password, &defaultHashParams)

	return hashedPassword, err
}

func VerifyPassword(hashedPassword, password string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hashedPassword)

	return match, err
}
