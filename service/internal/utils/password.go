package utils

import (
	"runtime"
	"github.com/alexedwards/argon2id"
	log "github.com/sirupsen/logrus"
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

	if err != nil {
		log.Errorf("Failed to hash password: %v", err)
		return "", err
	}

	return hashedPassword, nil
}

func VerifyPassword(hashedPassword, password string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hashedPassword)
	if err != nil {
		return false, err
	}
	return match, nil
}
