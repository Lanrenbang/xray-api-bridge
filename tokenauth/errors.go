package tokenauth

import "errors"

var (
	// ErrInvalidFormat indicates the token blob is malformed (bad base64 or too short).
	ErrInvalidFormat = errors.New("token: invalid format")

	// ErrInvalidSign indicates decryption failed (wrong key or tampered data).
	ErrInvalidSign = errors.New("token: invalid signature or tampered")

	// ErrTokenExpired indicates the token has passed its expiration time.
	ErrTokenExpired = errors.New("token: expired")

	// ErrInvalidPayload indicates the decrypted payload is not valid JSON.
	ErrInvalidPayload = errors.New("token: invalid payload")
)
