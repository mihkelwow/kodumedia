package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func randomID(prefix string) string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		sum := sha256.Sum256([]byte(prefix))
		copy(buffer, sum[:])
	}
	return prefix + "_" + hex.EncodeToString(buffer)
}

func stableID(prefix string, values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		_, _ = hash.Write([]byte(strings.ToLower(value)))
		_, _ = hash.Write([]byte{0})
	}
	return prefix + "_" + hex.EncodeToString(hash.Sum(nil)[:12])
}
