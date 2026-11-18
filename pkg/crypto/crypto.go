// Package crypto provides the shared AES-256-GCM envelope AIRA services use
// to protect tenant BYOK secrets at rest (see RFC-001 §9.1).
//
// Every service loads the same key material and encrypts/decrypts through
// this package, so a secret written by one service can be read by another.
// Keys are addressed by version: encryption always uses the highest version
// provided, while decryption uses the version recorded alongside each
// ciphertext — so a new key can be introduced for encryption while old
// ciphertext stays readable (zero-downtime rotation).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
)

// KeySize is the required key length, in bytes, for AES-256.
const KeySize = 32

var (
	// ErrInvalidKey is returned when a key is not exactly KeySize bytes.
	ErrInvalidKey = errors.New("crypto: key must be 32 bytes (AES-256)")
	// ErrUnknownKeyVersion is returned when decryption is asked for a key
	// version the Cipher was not built with.
	ErrUnknownKeyVersion = errors.New("crypto: unknown key version")
)

// Cipher encrypts and decrypts secrets with AES-256-GCM.
type Cipher struct {
	keys    map[int]cipher.AEAD
	primary int
}

// New builds a Cipher from a version -> key map. Every key must be KeySize
// bytes and at least one is required. The highest version is used for
// encryption.
func New(keys map[int][]byte) (*Cipher, error) {
	if len(keys) == 0 {
		return nil, errors.New("crypto: at least one key is required")
	}

	aeds := make(map[int]cipher.AEAD, len(keys))
	versions := make([]int, 0, len(keys))
	for v, key := range keys {
		if len(key) != KeySize {
			return nil, fmt.Errorf("%w (version %d has %d bytes)", ErrInvalidKey, v, len(key))
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("crypto: new cipher (version %d): %w", v, err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("crypto: new gcm (version %d): %w", v, err)
		}
		aeds[v] = aead
		versions = append(versions, v)
	}

	sort.Ints(versions)
	return &Cipher{keys: aeds, primary: versions[len(versions)-1]}, nil
}

// NewFromKey builds a single-key Cipher at version 1.
func NewFromKey(key []byte) (*Cipher, error) {
	return New(map[int][]byte{1: key})
}

// PrimaryVersion returns the key version used for encryption.
func (c *Cipher) PrimaryVersion() int { return c.primary }

// Encrypt seals plaintext under the primary key, returning the ciphertext,
// the random nonce it used, and the key version required to decrypt it.
func (c *Cipher) Encrypt(plaintext []byte) (ciphertext, nonce []byte, keyVersion int, err error) {
	aead := c.keys[c.primary]

	nonce = make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, 0, fmt.Errorf("crypto: read nonce: %w", err)
	}
	return aead.Seal(nil, nonce, plaintext, nil), nonce, c.primary, nil
}

// EncryptString is Encrypt for a string secret.
func (c *Cipher) EncryptString(plaintext string) (ciphertext, nonce []byte, keyVersion int, err error) {
	return c.Encrypt([]byte(plaintext))
}

// Decrypt reverses Encrypt using the key version recorded with the ciphertext.
func (c *Cipher) Decrypt(ciphertext, nonce []byte, keyVersion int) ([]byte, error) {
	aead, ok := c.keys[keyVersion]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrUnknownKeyVersion, keyVersion)
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("crypto: nonce is %d bytes, want %d", len(nonce), aead.NonceSize())
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decrypt: %w", err)
	}
	return plaintext, nil
}

// DecryptString is Decrypt for a string secret.
func (c *Cipher) DecryptString(ciphertext, nonce []byte, keyVersion int) (string, error) {
	b, err := c.Decrypt(ciphertext, nonce, keyVersion)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ParseKey decodes a KeySize-byte key from a hex or base64 string. Hex is
// tried first, then standard and URL-safe base64 (padded or raw).
func ParseKey(s string) ([]byte, error) {
	if b, err := hex.DecodeString(s); err == nil && len(b) == KeySize {
		return b, nil
	}
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil && len(b) == KeySize {
			return b, nil
		}
	}
	return nil, fmt.Errorf("%w: decoded key is not %d bytes", ErrInvalidKey, KeySize)
}
