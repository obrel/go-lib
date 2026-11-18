package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"
)

func randomKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, KeySize)
	if _, err := rand.Read(k); err != nil {
		t.Fatalf("rand key: %v", err)
	}
	return k
}

func newCipher(t *testing.T) (*Cipher, []byte) {
	t.Helper()
	key := randomKey(t)
	c, err := NewFromKey(key)
	if err != nil {
		t.Fatalf("NewFromKey: %v", err)
	}
	return c, key
}

func TestRoundTrip(t *testing.T) {
	c, _ := newCipher(t)
	secret := "sk-super-secret-api-key"

	ct, nonce, ver, err := c.EncryptString(secret)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ver != 1 {
		t.Fatalf("key version = %d, want 1", ver)
	}
	if bytes.Contains(ct, []byte(secret)) {
		t.Fatal("ciphertext contains the plaintext")
	}

	got, err := c.DecryptString(ct, nonce, ver)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != secret {
		t.Fatalf("decrypted %q, want %q", got, secret)
	}
}

func TestNonceIsUniquePerEncryption(t *testing.T) {
	c, _ := newCipher(t)

	ct1, nonce1, _, _ := c.EncryptString("same")
	ct2, nonce2, _, _ := c.EncryptString("same")

	if bytes.Equal(nonce1, nonce2) {
		t.Fatal("two encryptions reused the same nonce")
	}
	if bytes.Equal(ct1, ct2) {
		t.Fatal("two encryptions of the same plaintext produced identical ciphertext")
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	c1, _ := newCipher(t)
	c2, _ := newCipher(t)

	ct, nonce, ver, _ := c1.EncryptString("secret")
	if _, err := c2.Decrypt(ct, nonce, ver); err == nil {
		t.Fatal("expected decryption with the wrong key to fail")
	}
}

func TestDecryptTamperedCiphertextFails(t *testing.T) {
	c, _ := newCipher(t)

	ct, nonce, ver, _ := c.EncryptString("secret")
	ct[len(ct)-1] ^= 0xff

	if _, err := c.Decrypt(ct, nonce, ver); err == nil {
		t.Fatal("expected decryption of tampered ciphertext to fail")
	}
}

func TestUnknownKeyVersion(t *testing.T) {
	c, _ := newCipher(t)

	ct, nonce, _, _ := c.EncryptString("secret")
	_, err := c.Decrypt(ct, nonce, 99)
	if !errors.Is(err, ErrUnknownKeyVersion) {
		t.Fatalf("err = %v, want ErrUnknownKeyVersion", err)
	}
}

func TestNewRejectsInvalidKeySize(t *testing.T) {
	if _, err := NewFromKey([]byte("too-short")); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("err = %v, want ErrInvalidKey", err)
	}
	if _, err := New(map[int][]byte{}); err == nil {
		t.Fatal("expected an error for an empty key map")
	}
}

func TestKeyRotation(t *testing.T) {
	k1, k2 := randomKey(t), randomKey(t)

	// Encrypt under an old single-key cipher (version 1)...
	old, err := NewFromKey(k1)
	if err != nil {
		t.Fatalf("NewFromKey: %v", err)
	}
	ct, nonce, ver, err := old.EncryptString("legacy secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ver != 1 {
		t.Fatalf("key version = %d, want 1", ver)
	}

	// ...then a cipher trusting both keys must still decrypt it, and must
	// now encrypt new data under version 2.
	rotated, err := New(map[int][]byte{1: k1, 2: k2})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if rotated.PrimaryVersion() != 2 {
		t.Fatalf("primary version = %d, want 2", rotated.PrimaryVersion())
	}
	got, err := rotated.DecryptString(ct, nonce, ver)
	if err != nil {
		t.Fatalf("Decrypt old: %v", err)
	}
	if got != "legacy secret" {
		t.Fatalf("decrypted %q, want %q", got, "legacy secret")
	}

	if _, _, newVer, _ := rotated.EncryptString("new secret"); newVer != 2 {
		t.Fatalf("new ciphertext version = %d, want 2", newVer)
	}
}

func TestParseKey(t *testing.T) {
	key := randomKey(t)

	hexKey := hex.EncodeToString(key)
	if got, err := ParseKey(hexKey); err != nil || !bytes.Equal(got, key) {
		t.Fatalf("ParseKey(hex) = %x, %v", got, err)
	}

	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if got, err := ParseKey(enc.EncodeToString(key)); err != nil || !bytes.Equal(got, key) {
			t.Fatalf("ParseKey(base64) = %x, %v", got, err)
		}
	}

	if _, err := ParseKey("not-a-key"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("err = %v, want ErrInvalidKey", err)
	}
}
