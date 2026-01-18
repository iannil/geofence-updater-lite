// Package crypto provides Ed25519 digital signature functionality
// for signing and verifying geofence data.
package crypto

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	// PublicKeySize is the size of an Ed25519 public key in bytes.
	PublicKeySize = ed25519.PublicKeySize

	// PrivateKeySize is the size of an Ed25519 private key in bytes.
	PrivateKeySize = ed25519.PrivateKeySize

	// SignatureSize is the size of an Ed25519 signature in bytes.
	SignatureSize = ed25519.SignatureSize

	// KeyIDSize is the size of the key identifier hash.
	KeyIDSize = 8
)

// KeyPair represents an Ed25519 key pair.
type KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
	KeyID      string
}

// GenerateKeyPair generates a new Ed25519 key pair.
func GenerateKeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	keyID := computeKeyID(publicKey)

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		KeyID:      keyID,
	}, nil
}

// GenerateKeyFromReader generates a new Ed25519 key pair using a specific reader.
func GenerateKeyFromReader(r io.Reader) (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(r)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	keyID := computeKeyID(publicKey)

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		KeyID:      keyID,
	}, nil
}

// DeriveKeyPair creates a KeyPair from existing keys.
func DeriveKeyPair(publicKey, privateKey []byte) (*KeyPair, error) {
	if len(publicKey) != PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d", len(publicKey))
	}
	if len(privateKey) != PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: %d", len(privateKey))
	}

	// Verify the key pair is valid by deriving the public key from private key
	// and comparing it byte-by-byte (Equal can be finicky)
	derivedPublicKey := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
	if len(derivedPublicKey) != len(publicKey) {
		return nil, fmt.Errorf("public key does not match private key: derived len=%d, input len=%d", len(derivedPublicKey), len(publicKey))
	}
	// Manual byte comparison
	for i := 0; i < len(derivedPublicKey); i++ {
		if derivedPublicKey[i] != publicKey[i] {
			return nil, fmt.Errorf("public key does not match private key: mismatch at byte %d", i)
		}
	}

	keyID := computeKeyID(publicKey)

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		KeyID:      keyID,
	}, nil
}

// PublicKeyFromBytes creates a KeyPair with only a public key (for verification).
func PublicKeyFromBytes(publicKey []byte) (*KeyPair, error) {
	if len(publicKey) != PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d", len(publicKey))
	}

	keyID := computeKeyID(publicKey)

	return &KeyPair{
		PublicKey: publicKey,
		KeyID:     keyID,
	}, nil
}

// computeKeyID generates a short identifier for a public key.
func computeKeyID(publicKey []byte) string {
	h := sha256.New()
	h.Write(publicKey)
	hash := h.Sum(nil)
	return hex.EncodeToString(hash[:KeyIDSize])
}

// Sign signs a message with the private key.
func (k *KeyPair) Sign(message []byte) []byte {
	if len(k.PrivateKey) == 0 {
		panic("private key not available")
	}
	return ed25519.Sign(k.PrivateKey, message)
}

// Verify verifies a signature against a message.
func (k *KeyPair) Verify(message, signature []byte) bool {
	if len(k.PublicKey) != PublicKeySize {
		return false
	}
	return ed25519.Verify(k.PublicKey, message, signature)
}

// Sign signs a message with a private key.
func Sign(privateKey, message []byte) []byte {
	if len(privateKey) != PrivateKeySize {
		panic("invalid private key size")
	}
	return ed25519.Sign(privateKey, message)
}

// Verify verifies a signature with a public key.
func Verify(publicKey, message, signature []byte) bool {
	if len(publicKey) != PublicKeySize {
		return false
	}
	if len(signature) != SignatureSize {
		return false
	}
	return ed25519.Verify(publicKey, message, signature)
}

// MarshalPublicKeyHex encodes a public key as a hex string.
func MarshalPublicKeyHex(publicKey []byte) string {
	return hex.EncodeToString(publicKey)
}

// MarshalPrivateKeyHex encodes a private key as a hex string.
func MarshalPrivateKeyHex(privateKey []byte) string {
	return hex.EncodeToString(privateKey)
}

// UnmarshalPublicKeyHex decodes a hex string to a public key.
func UnmarshalPublicKeyHex(s string) ([]byte, error) {
	key, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex encoding: %w", err)
	}
	if len(key) != PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d", len(key))
	}
	return key, nil
}

// UnmarshalPrivateKeyHex decodes a hex string to a private key.
func UnmarshalPrivateKeyHex(s string) ([]byte, error) {
	key, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex encoding: %w", err)
	}
	if len(key) != PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: %d", len(key))
	}
	return key, nil
}

// PublicKeyToKeyID computes the key ID for a public key.
func PublicKeyToKeyID(publicKey []byte) (string, error) {
	if len(publicKey) != PublicKeySize {
		return "", fmt.Errorf("invalid public key size: %d", len(publicKey))
	}
	return computeKeyID(publicKey), nil
}

// ComputeSHA256 computes the SHA-256 hash of data.
func ComputeSHA256(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// VerifyHash verifies that data matches the expected SHA-256 hash.
func VerifyHash(data, expectedHash []byte) bool {
	computed := ComputeSHA256(data)
	return bytes.Equal(computed, expectedHash)
}
