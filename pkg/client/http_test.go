package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/config"
	"github.com/iannil/geofence-updater-lite/pkg/crypto"
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
)

func testClientConfig(t *testing.T, serverURL string) *config.ClientConfig {
	t.Helper()
	return &config.ClientConfig{
		ManifestURL:        serverURL + "/manifest.json",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}
}

func TestNewClient(t *testing.T) {
	cfg := &config.ClientConfig{
		ManifestURL:        "https://example.com/manifest.json",
		HTTPTimeout:        30 * time.Second,
		UserAgent:          "test-agent/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.userAgent != cfg.UserAgent {
		t.Errorf("userAgent = %s, want %s", client.userAgent, cfg.UserAgent)
	}
}

func TestNewClient_WithPublicKey(t *testing.T) {
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	cfg := &config.ClientConfig{
		ManifestURL:  "https://example.com/manifest.json",
		HTTPTimeout:  30 * time.Second,
		UserAgent:    "test-agent/1.0",
		StorePath:    filepath.Join(t.TempDir(), "client.db"),
		PublicKeyHex: crypto.MarshalPublicKeyHex(kp.PublicKey),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if len(client.publicKey) != crypto.PublicKeySize {
		t.Errorf("publicKey size = %d, want %d", len(client.publicKey), crypto.PublicKeySize)
	}
}

func TestNewClient_InvalidPublicKey(t *testing.T) {
	cfg := &config.ClientConfig{
		ManifestURL:  "https://example.com/manifest.json",
		HTTPTimeout:  30 * time.Second,
		UserAgent:    "test-agent/1.0",
		StorePath:    filepath.Join(t.TempDir(), "client.db"),
		PublicKeyHex: "invalid-hex!",
	}

	_, err := NewClient(cfg)
	if err == nil {
		t.Error("expected error for invalid public key")
	}
}

func TestFetchManifest_Success(t *testing.T) {
	manifest := &geofence.Manifest{
		Version:     1,
		Timestamp:   time.Now().Unix(),
		SnapshotURL: "/snapshot.bin",
		Message:     "Test manifest",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL + "/manifest.json",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	result, err := client.FetchManifest(ctx)
	if err != nil {
		t.Fatalf("FetchManifest failed: %v", err)
	}

	if result.Version != manifest.Version {
		t.Errorf("Version = %d, want %d", result.Version, manifest.Version)
	}
	if result.Message != manifest.Message {
		t.Errorf("Message = %s, want %s", result.Message, manifest.Message)
	}
}

func TestFetchManifest_SignatureVerification(t *testing.T) {
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	manifest := &geofence.Manifest{
		Version:     2,
		Timestamp:   time.Now().Unix(),
		SnapshotURL: "/snapshot.bin",
		Message:     "Signed manifest",
	}

	// Sign the manifest
	manifestData, err := manifest.MarshalBinaryForSigning()
	if err != nil {
		t.Fatalf("MarshalBinaryForSigning failed: %v", err)
	}
	signature, err := kp.Sign(manifestData)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	manifest.SetSignature(signature, kp.KeyID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:  server.URL + "/manifest.json",
		HTTPTimeout:  5 * time.Second,
		UserAgent:    "test/1.0",
		StorePath:    filepath.Join(t.TempDir(), "client.db"),
		PublicKeyHex: crypto.MarshalPublicKeyHex(kp.PublicKey),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	result, err := client.FetchManifest(ctx)
	if err != nil {
		t.Fatalf("FetchManifest failed: %v", err)
	}

	if result.Version != manifest.Version {
		t.Errorf("Version = %d, want %d", result.Version, manifest.Version)
	}
}

func TestFetchManifest_NoSignature(t *testing.T) {
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	manifest := &geofence.Manifest{
		Version:   1,
		Timestamp: time.Now().Unix(),
		// No signature
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:  server.URL + "/manifest.json",
		HTTPTimeout:  5 * time.Second,
		UserAgent:    "test/1.0",
		StorePath:    filepath.Join(t.TempDir(), "client.db"),
		PublicKeyHex: crypto.MarshalPublicKeyHex(kp.PublicKey),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	_, err = client.FetchManifest(ctx)
	if err == nil {
		t.Error("expected error for missing signature")
	}
}

func TestFetchManifest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL + "/manifest.json",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	_, err = client.FetchManifest(ctx)
	if err == nil {
		t.Error("expected error for server error")
	}
}

func TestFetchManifest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL + "/manifest.json",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	_, err = client.FetchManifest(ctx)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFetchSnapshot_Success(t *testing.T) {
	snapshotData := []byte("binary snapshot data here")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/snapshot.bin" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(snapshotData)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL,
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	result, err := client.FetchSnapshot(ctx, "/snapshot.bin")
	if err != nil {
		t.Fatalf("FetchSnapshot failed: %v", err)
	}

	if string(result) != string(snapshotData) {
		t.Errorf("data mismatch")
	}
}

func TestFetchSnapshot_EmptyURL(t *testing.T) {
	cfg := &config.ClientConfig{
		ManifestURL:        "https://example.com",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	_, err = client.FetchSnapshot(ctx, "")
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestFetchDelta_Success(t *testing.T) {
	deltaData := []byte("binary delta data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(deltaData)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL,
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	result, err := client.FetchDelta(ctx, "/delta.bin")
	if err != nil {
		t.Fatalf("FetchDelta failed: %v", err)
	}

	if string(result) != string(deltaData) {
		t.Errorf("data mismatch")
	}
}

func TestFetchDelta_EmptyURL(t *testing.T) {
	cfg := &config.ClientConfig{
		ManifestURL:        "https://example.com",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	_, err = client.FetchDelta(ctx, "")
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestFetchWithProgress(t *testing.T) {
	data := []byte("test data for progress tracking")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "31")
		w.Write(data)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL,
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	var progressCalled bool
	ctx := context.Background()
	result, err := client.FetchWithProgress(ctx, "/data.bin", func(downloaded, total int64) {
		progressCalled = true
	})
	if err != nil {
		t.Fatalf("FetchWithProgress failed: %v", err)
	}

	if !progressCalled {
		t.Error("progress callback was not called")
	}
	if string(result) != string(data) {
		t.Errorf("data mismatch")
	}
}

func TestFetchManifestWithRetry_Success(t *testing.T) {
	attempts := 0
	manifest := &geofence.Manifest{
		Version:   1,
		Timestamp: time.Now().Unix(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL + "/manifest.json",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	result, err := client.FetchManifestWithRetry(ctx, 3)
	if err != nil {
		t.Fatalf("FetchManifestWithRetry failed: %v", err)
	}

	if attempts < 2 {
		t.Errorf("expected at least 2 attempts, got %d", attempts)
	}
	if result.Version != manifest.Version {
		t.Errorf("Version = %d, want %d", result.Version, manifest.Version)
	}
}

func TestFetchManifestWithRetry_AllFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL + "/manifest.json",
		HTTPTimeout:        1 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	_, err = client.FetchManifestWithRetry(ctx, 2)
	if err == nil {
		t.Error("expected error after all retries fail")
	}
}

func TestFetchManifestWithRetry_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL + "/manifest.json",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err = client.FetchManifestWithRetry(ctx, 5)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
}

func TestGetLastModified(t *testing.T) {
	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("expected HEAD, got %s", r.Method)
		}
		w.Header().Set("Last-Modified", expectedTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL + "/manifest.json",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	result, err := client.GetLastModified(ctx)
	if err != nil {
		t.Fatalf("GetLastModified failed: %v", err)
	}

	if !result.Equal(expectedTime) {
		t.Errorf("time = %v, want %v", result, expectedTime)
	}
}

func TestGetLastModified_NoHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.ClientConfig{
		ManifestURL:        server.URL + "/manifest.json",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test/1.0",
		StorePath:          filepath.Join(t.TempDir(), "client.db"),
		InsecureSkipVerify: true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx := context.Background()
	result, err := client.GetLastModified(ctx)
	if err != nil {
		t.Fatalf("GetLastModified failed: %v", err)
	}

	if !result.IsZero() {
		t.Errorf("expected zero time, got %v", result)
	}
}

func TestVerifyDeltaHash(t *testing.T) {
	data := []byte("test delta data")
	hash := crypto.ComputeSHA256(data)

	if !VerifyDeltaHash(data, hash) {
		t.Error("hash verification failed for matching data")
	}

	if VerifyDeltaHash(data, []byte("wronghash")) {
		t.Error("hash verification should fail for wrong hash")
	}

	// Empty hash should return true (no verification needed)
	if !VerifyDeltaHash(data, nil) {
		t.Error("empty hash should return true")
	}
}

func TestIsAbsoluteURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/path", true},
		{"http://example.com", true},
		{"/path/to/file", false},
		{"relative/path", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isAbsoluteURL(tt.url)
		if result != tt.expected {
			t.Errorf("isAbsoluteURL(%s) = %v, want %v", tt.url, result, tt.expected)
		}
	}
}
