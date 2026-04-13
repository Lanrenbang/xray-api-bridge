package tokenauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// payload is the internal structure encrypted inside the token.
type payload struct {
	UUID string `json:"uuid"`
	Exp  int64  `json:"exp"`
}

// GenerateEncryptedToken creates an AES-256-GCM encrypted token containing the
// given uuid. The token format is: base64url( IV(12) + ciphertext + AuthTag(16) ).
func GenerateEncryptedToken(secret, uuid string, ttl time.Duration) (string, error) {
	p := payload{
		UUID: uuid,
		Exp:  time.Now().Unix() + int64(ttl.Seconds()),
	}

	plaintext, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}

	gcm, err := newGCM(secret)
	if err != nil {
		return "", err
	}

	// Random IV (GCM standard 12 bytes)
	iv := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("random iv: %w", err)
	}

	// Seal appends ciphertext + AuthTag
	ciphertext := gcm.Seal(nil, iv, plaintext, nil)

	// Concatenate: IV + ciphertext + AuthTag → base64url
	blob := make([]byte, 0, len(iv)+len(ciphertext))
	blob = append(blob, iv...)
	blob = append(blob, ciphertext...)

	return base64.RawURLEncoding.EncodeToString(blob), nil
}

// VerifyEncryptedToken decrypts the token and returns the embedded uuid.
// Returns an error if the token is malformed, expired, or tampered with.
func VerifyEncryptedToken(token, secret string) (string, error) {
	blob, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", ErrInvalidFormat
	}

	gcm, err := newGCM(secret)
	if err != nil {
		return "", ErrInvalidFormat
	}

	nonceSize := gcm.NonceSize() // 12
	if len(blob) < nonceSize+1+gcm.Overhead() {
		return "", ErrInvalidFormat
	}
	iv := blob[:nonceSize]
	ciphertext := blob[nonceSize:]

	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return "", ErrInvalidSign
	}

	var p payload
	if err := json.Unmarshal(plaintext, &p); err != nil {
		return "", ErrInvalidPayload
	}

	if p.Exp > 0 && p.Exp < time.Now().Unix() {
		return "", ErrTokenExpired
	}

	return p.UUID, nil
}

// newGCM creates an AES-256-GCM cipher from an arbitrary-length secret.
func newGCM(secret string) (cipher.AEAD, error) {
	key := deriveKey(secret)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	return gcm, nil
}

// deriveKey derives a 32-byte AES key from an arbitrary-length secret via SHA-256.
func deriveKey(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}
