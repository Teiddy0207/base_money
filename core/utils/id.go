package utils

import (
	"crypto/rand"
	"encoding/base64"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

func GenerateID() string {
	id, err := gonanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", 7)
	if err != nil {
		return ""
	}
	return id
}

// GenerateRandomString generates a cryptographically secure random string
func GenerateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to nanoid if crypto/rand fails
		id, _ := gonanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", length)
		return id
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}
