package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Hasher hashes and verifies passwords with Argon2id (the current OWASP
// recommendation). Hashes are stored in the standard PHC string format, e.g.
//
//	$argon2id$v=19$m=65536,t=3,p=2$<saltB64>$<hashB64>
//
// so the parameters travel with the hash and old hashes stay verifiable even if
// the defaults change later.
type Hasher struct {
	memory  uint32
	time    uint32
	threads uint8
	keyLen  uint32
	saltLen uint32
}

// NewHasher builds a Hasher from the configured cost parameters.
func NewHasher(memoryKB, time uint32, threads uint8) *Hasher {
	return &Hasher{memory: memoryKB, time: time, threads: threads, keyLen: 32, saltLen: 16}
}

// Hash derives a PHC-encoded Argon2id hash of the password.
func (h *Hasher) Hash(password string) (string, error) {
	salt := make([]byte, h.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, h.time, h.memory, h.threads, h.keyLen)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.memory, h.time, h.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches a PHC-encoded Argon2id hash.
// It re-derives the key using the parameters embedded in the encoded hash and
// compares in constant time. A malformed encoded string returns an error.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=...,t=...,p=...", "<salt>", "<hash>"]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("password: unrecognised hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("password: version: %w", err)
	}

	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, fmt.Errorf("password: params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("password: salt: %w", err)
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("password: hash: %w", err)
	}

	got := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
