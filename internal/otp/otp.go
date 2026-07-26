// Package otp generates one-time passcodes and hashes them for storage.
//
// OTPs are high-entropy random strings, so a salted PBKDF2-HMAC-SHA256 hash
// plus the per-share attempt lockout is more than sufficient. Never store or
// log the plaintext OTP; only Hash's output goes to the database.
package otp

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"math/big"

	"golang.org/x/crypto/pbkdf2"
)

const (
	iterations = 100_000
	keyLen     = 32
	saltLen    = 16
)

// Generate returns a random string of the given length drawn from alphabet.
func Generate(length int, alphabet string) (string, error) {
	if length <= 0 || alphabet == "" {
		return "", nil
	}
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}

// Hash derives a salted PBKDF2 hash of the OTP. Returns hex(hash), hex(salt).
func Hash(code string) (hashHex, saltHex string, err error) {
	salt := make([]byte, saltLen)
	if _, err = rand.Read(salt); err != nil {
		return "", "", err
	}
	dk := pbkdf2.Key([]byte(code), salt, iterations, keyLen, sha256.New)
	return hex.EncodeToString(dk), hex.EncodeToString(salt), nil
}

// Verify checks a candidate OTP against the stored hash+salt in constant time.
func Verify(candidate, hashHex, saltHex string) bool {
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return false
	}
	want, err := hex.DecodeString(hashHex)
	if err != nil {
		return false
	}
	got := pbkdf2.Key([]byte(candidate), salt, iterations, keyLen, sha256.New)
	return subtle.ConstantTimeCompare(got, want) == 1
}
