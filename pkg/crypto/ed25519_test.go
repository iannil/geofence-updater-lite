package crypto

import (
	"bytes"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if len(kp.PublicKey) != PublicKeySize {
		t.Errorf("public key size = %d, want %d", len(kp.PublicKey), PublicKeySize)
	}

	if len(kp.PrivateKey) != PrivateKeySize {
		t.Errorf("private key size = %d, want %d", len(kp.PrivateKey), PrivateKeySize)
	}

	if kp.KeyID == "" {
		t.Error("key ID is empty")
	}
}

func TestSignAndVerify(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	message := []byte("test message for geofence data")

	signature, err := kp.Sign(message)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if len(signature) != SignatureSize {
		t.Errorf("signature size = %d, want %d", len(signature), SignatureSize)
	}

	if !kp.Verify(message, signature) {
		t.Error("signature verification failed")
	}

	// Wrong message should fail
	if kp.Verify([]byte("wrong message"), signature) {
		t.Error("signature verification should fail for wrong message")
	}
}

func TestVerifyInvalid(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	message := []byte("test message")
	wrongSignature := make([]byte, SignatureSize)

	if kp.Verify(message, wrongSignature) {
		t.Error("wrong signature should not verify")
	}
}

func TestKeyPairMethods(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	message := []byte("test message")

	// Test using global Sign function
	sig1, err := Sign(kp.PrivateKey, message)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if !Verify(kp.PublicKey, message, sig1) {
		t.Error("global Sign/Verify failed")
	}

	// Test using KeyPair method
	sig2, err := kp.Sign(message)
	if err != nil {
		t.Fatalf("KeyPair.Sign failed: %v", err)
	}
	if !kp.Verify(message, sig2) {
		t.Error("KeyPair.Sign/Verify failed")
	}

	// Signatures should be the same for the same message
	if !bytes.Equal(sig1, sig2) {
		t.Error("signatures differ for same message")
	}
}

func TestMarshalHex(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	pubHex := MarshalPublicKeyHex(kp.PublicKey)
	privHex := MarshalPrivateKeyHex(kp.PrivateKey)

	pubDecoded, err := UnmarshalPublicKeyHex(pubHex)
	if err != nil {
		t.Fatalf("UnmarshalPublicKeyHex failed: %v", err)
	}

	privDecoded, err := UnmarshalPrivateKeyHex(privHex)
	if err != nil {
		t.Fatalf("UnmarshalPrivateKeyHex failed: %v", err)
	}

	message := []byte("test message")
	sig, err := kp.Sign(message)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if !Verify(pubDecoded, message, sig) {
		t.Error("verification failed after round-trip encoding")
	}

	// Verify private key still works
	sig2, err := Sign(privDecoded, message)
	if err != nil {
		t.Fatalf("Sign with decoded key failed: %v", err)
	}
	if !Verify(pubDecoded, message, sig2) {
		t.Error("verification failed with decoded private key")
	}
}

func TestUnmarshalInvalidHex(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		unmarshal func(string) ([]byte, error)
	}{
		{"invalid public hex", "not-hex!", UnmarshalPublicKeyHex},
		{"invalid private hex", "not-hex!", UnmarshalPrivateKeyHex},
		{"short public hex", "abc123", UnmarshalPublicKeyHex},
		{"short private hex", "abc123", UnmarshalPrivateKeyHex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.unmarshal(tt.hex)
			if err == nil {
				t.Error("expected error for invalid hex")
			}
		})
	}
}

func TestDeriveKeyPair(t *testing.T) {
	kp1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Debug: print key lengths
	t.Logf("kp1.PublicKey len: %d", len(kp1.PublicKey))
	t.Logf("kp1.PrivateKey len: %d", len(kp1.PrivateKey))

	// The KeyPair from GenerateKeyPair already has matching keys
	// DeriveKeyPair should work with them
	kp2, err := DeriveKeyPair(kp1.PublicKey, kp1.PrivateKey)
	if err != nil {
		t.Fatalf("DeriveKeyPair failed: %v", err)
	}

	if kp1.KeyID != kp2.KeyID {
		t.Errorf("key IDs differ: %s vs %s", kp1.KeyID, kp2.KeyID)
	}

	message := []byte("test")
	sig1, err := kp1.Sign(message)
	if err != nil {
		t.Fatalf("kp1.Sign failed: %v", err)
	}
	sig2, err := kp2.Sign(message)
	if err != nil {
		t.Fatalf("kp2.Sign failed: %v", err)
	}

	if !bytes.Equal(sig1, sig2) {
		t.Error("signatures differ for derived key pair")
	}
}

func TestPublicKeyFromBytes(t *testing.T) {
	kp1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	message := []byte("test message")
	signature, err := kp1.Sign(message)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Create verification-only key pair
	kp2, err := PublicKeyFromBytes(kp1.PublicKey)
	if err != nil {
		t.Fatalf("PublicKeyFromBytes failed: %v", err)
	}

	if kp2.PrivateKey != nil {
		t.Error("private key should be nil")
	}

	if !kp2.Verify(message, signature) {
		t.Error("verification failed with public key only")
	}
}

func TestPublicKeyToKeyID(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	keyID, err := PublicKeyToKeyID(kp.PublicKey)
	if err != nil {
		t.Fatalf("PublicKeyToKeyID failed: %v", err)
	}

	if keyID != kp.KeyID {
		t.Errorf("key IDs differ: %s vs %s", keyID, kp.KeyID)
	}
}

func TestInvalidKeySizes(t *testing.T) {
	t.Run("invalid public key size", func(t *testing.T) {
		_, err := PublicKeyFromBytes([]byte("short"))
		if err == nil {
			t.Error("expected error for short public key")
		}
	})

	t.Run("invalid private key size", func(t *testing.T) {
		_, err := DeriveKeyPair(make([]byte, PublicKeySize), []byte("short"))
		if err == nil {
			t.Error("expected error for short private key")
		}
	})

	t.Run("mismatched key pair", func(t *testing.T) {
		kp1, _ := GenerateKeyPair()
		kp2, _ := GenerateKeyPair()
		_, err := DeriveKeyPair(kp1.PublicKey, kp2.PrivateKey)
		if err == nil {
			t.Error("expected error for mismatched key pair")
		}
	})
}
