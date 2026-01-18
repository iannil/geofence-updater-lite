// Package client provides HTTP client functionality for downloading
// geofence updates from a CDN or static file server.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/config"
	"github.com/iannil/geofence-updater-lite/pkg/crypto"
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
)

// Client is an HTTP client for downloading geofence updates.
type Client struct {
	httpClient *http.Client
	userAgent  string
	publicKey  []byte
	cdnBaseURL string
}

// NewClient creates a new HTTP client for geofence updates.
func NewClient(cfg *config.ClientConfig) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Decode public key from hex
	var publicKey []byte
	if cfg.PublicKeyHex != "" {
		var err error
		publicKey, err = crypto.UnmarshalPublicKeyHex(cfg.PublicKeyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid public key: %w", err)
		}
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
			},
		},
		userAgent:  cfg.UserAgent,
		publicKey:  publicKey,
		cdnBaseURL: cfg.ManifestURL,
	}, nil
}

// FetchManifest downloads and verifies the manifest from the remote server.
func (c *Client) FetchManifest(ctx context.Context) (*geofence.Manifest, error) {
	manifestURL := c.cdnBaseURL
	if manifestURL == "" {
		return nil, fmt.Errorf("no CDN base URL configured")
	}

	// Ensure path ends with /manifest.json
	if u, err := url.Parse(manifestURL); err == nil {
		if u.Path == "" || u.Path == "/" {
			manifestURL = path.Join(manifestURL, "manifest.json")
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Check content length
	if resp.ContentLength > config.DefaultMaxDownloadSize {
		return nil, fmt.Errorf("manifest too large: %d bytes", resp.ContentLength)
	}

	// Read response body
	manifestData, err := io.ReadAll(io.LimitReader(resp.Body, config.DefaultMaxDownloadSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	// Parse manifest
	var manifest geofence.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Verify manifest signature
	if err := c.verifyManifestSignature(&manifest, manifestData); err != nil {
		return nil, fmt.Errorf("manifest signature verification failed: %w", err)
	}

	return &manifest, nil
}

// verifyManifestSignature verifies the signature of a manifest.
func (c *Client) verifyManifestSignature(manifest *geofence.Manifest, data []byte) error {
	if len(c.publicKey) == 0 {
		// No public key configured, skip verification
		return nil
	}

	if len(manifest.Signature) == 0 {
		return fmt.Errorf("manifest has no signature")
	}

	// Verify the signature
	if !crypto.Verify(c.publicKey, data, manifest.Signature) {
		return fmt.Errorf("invalid signature")
	}

	// Verify KeyID matches if specified
	if manifest.KeyID != "" {
		expectedKeyID, err := crypto.PublicKeyToKeyID(c.publicKey)
		if err != nil {
			return fmt.Errorf("failed to compute key ID: %w", err)
		}
		if expectedKeyID != manifest.KeyID {
			return fmt.Errorf("key ID mismatch: expected %s, got %s", expectedKeyID, manifest.KeyID)
		}
	}

	return nil
}

// FetchSnapshot downloads the snapshot file from the remote server.
func (c *Client) FetchSnapshot(ctx context.Context, snapshotURL string) ([]byte, error) {
	if snapshotURL == "" {
		return nil, fmt.Errorf("empty snapshot URL")
	}

	// Resolve relative URL
	fullURL := snapshotURL
	if !isAbsoluteURL(snapshotURL) {
		fullURL = c.cdnBaseURL + snapshotURL
	}

	return c.fetchBinary(ctx, fullURL, "snapshot")
}

// FetchDelta downloads the delta file from the remote server.
func (c *Client) FetchDelta(ctx context.Context, deltaURL string) ([]byte, error) {
	if deltaURL == "" {
		return nil, fmt.Errorf("empty delta URL")
	}

	// Resolve relative URL
	fullURL := deltaURL
	if !isAbsoluteURL(deltaURL) {
		fullURL = c.cdnBaseURL + deltaURL
	}

	return c.fetchBinary(ctx, fullURL, "delta")
}

// fetchBinary downloads a binary file with size limit.
func (c *Client) fetchBinary(ctx context.Context, urlStr, fileType string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", fileType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for %s: %d", fileType, resp.StatusCode)
	}

	// Check content length
	if resp.ContentLength > config.DefaultMaxDownloadSize {
		return nil, fmt.Errorf("%s too large: %d bytes", fileType, resp.ContentLength)
	}

	// Read response body
	data, err := io.ReadAll(io.LimitReader(resp.Body, config.DefaultMaxDownloadSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", fileType, err)
	}

	return data, nil
}

// FetchWithProgress downloads a file with progress reporting.
func (c *Client) FetchWithProgress(ctx context.Context, urlStr string, onProgress ProgressFunc) ([]byte, error) {
	if !isAbsoluteURL(urlStr) {
		urlStr = c.cdnBaseURL + urlStr
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Get total size
	totalSize := resp.ContentLength
	if totalSize < 0 {
		totalSize = 0 // Unknown size
	}

	if totalSize > config.DefaultMaxDownloadSize {
		return nil, fmt.Errorf("file too large: %d bytes", totalSize)
	}

	// Read with progress reporting
	var buf bytes.Buffer
	_, err = io.Copy(&buf, &progressReader{
		reader:    resp.Body,
		total:     totalSize,
		onProgress: onProgress,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return buf.Bytes(), nil
}

// ProgressFunc is called during download with progress updates.
type ProgressFunc func(downloaded, total int64)

// progressReader wraps an io.Reader to report progress.
type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	onProgress ProgressFunc
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	if n > 0 {
		pr.downloaded += int64(n)
		if pr.onProgress != nil {
			pr.onProgress(pr.downloaded, pr.total)
		}
	}
	return
}

// isAbsoluteURL checks if a URL is absolute.
func isAbsoluteURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// VerifyDeltaHash verifies the SHA-256 hash of delta data.
func VerifyDeltaHash(data []byte, expectedHash []byte) bool {
	if len(expectedHash) == 0 {
		return true // No hash to verify
	}

	// Compute SHA-256
	hash := crypto.ComputeSHA256(data)

	return bytes.Equal(hash, expectedHash)
}

// FetchManifestWithRetry fetches the manifest with retry logic.
func (c *Client) FetchManifestWithRetry(ctx context.Context, maxRetries int) (*geofence.Manifest, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		manifest, err := c.FetchManifest(ctx)
		if err == nil {
			return manifest, nil
		}
		lastErr = err

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Wait before retry (exponential backoff)
		waitTime := time.Duration(1<<uint(i)) * time.Second
		if waitTime > 30*time.Second {
			waitTime = 30 * time.Second
		}
		time.Sleep(waitTime)
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// GetLastModified returns the Last-Modified header of the manifest.
func (c *Client) GetLastModified(ctx context.Context) (time.Time, error) {
	manifestURL := c.cdnBaseURL
	if u, err := url.Parse(manifestURL); err == nil {
		if u.Path == "" || u.Path == "/" {
			manifestURL = path.Join(manifestURL, "manifest.json")
		}
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", manifestURL, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to HEAD manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	lastModified := resp.Header.Get("Last-Modified")
	if lastModified == "" {
		return time.Time{}, nil
	}

	return http.ParseTime(lastModified)
}
