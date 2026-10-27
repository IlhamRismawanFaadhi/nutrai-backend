package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// Generate mengembalikan (plaintext, hash). Plaintext hanya ditampilkan sekali ke user.
func Generate() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain := "nutrai_" + hex.EncodeToString(b)
	return plain, Hash(plain), nil
}

func Hash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
