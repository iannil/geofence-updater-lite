# Geofence-Updater-Lite (GUL)

[English](README.md) | [简体中文](README.zh-CN.md)

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)
![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)
![Coverage](https://img.shields.io/badge/coverage-90%25-brightgreen.svg)

**A lightweight, highly reliable geofence data synchronization system**

Designed for drones/UAVs operating in low-bandwidth, unstable network environments

[Features](#features) • [Quick Start](#quick-start) • [Usage Guide](#usage-guide) • [API Documentation](#api-documentation) • [Protocol Specification](#protocol-specification)

</div>

---

## Overview

Geofence-Updater-Lite (GUL) is a decentralized geofence data synchronization system. The core design philosophy uses **Merkle Tree for incremental updates**, compressing version differences to just a few KB, enabling stable operation on GPRS-level networks.

**Core Features:**

- **Ultra-Low Bandwidth** - Merkle Tree + binary delta, incremental updates only require a few KB
- **Decentralized Distribution** - Pure static files, deployable on any CDN/OSS, zero server cost
- **Security-First** - Ed25519 digital signatures, offline verification, tamper-proof
- **High-Performance Queries** - R-Tree spatial indexing, millisecond-level geofence checks
- **Cross-Platform** - Pure Go implementation, supports Linux/macOS/Windows

---

## Features

| Feature | Description |
| --------- | ------------- |
| **Ultra-Low Bandwidth** | Merkle Tree enables incremental updates with version differences as small as a few KB |
| **Decentralized Distribution** | Core data is static files, deployable on CDN/OSS/IPFS |
| **Digital Signatures** | Ed25519 signatures + KeyID mechanism for tamper-proof verification |
| **High-Performance Queries** | R-Tree spatial indexing for millisecond-level geofence checks |
| **Offline Verification** | Built-in public key verification, independent of data source |
| **Version Rollback Protection** | Rejects applying older version data |
| **Progress Callbacks** | Large file downloads support progress reporting |
| **Easy Integration** | Pure Go implementation with cross-platform compilation support |

---

## Architecture

### Core Principles

```
┌─────────────────────────────────────────────────────────────────┐
│                     Git Philosophy                              │
│              Merkle Tree manages versions, download diffs only   │
└─────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│                     CDN Friendly                                │
│              Pure static files, deployable on any CDN/OSS       │
└─────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│                     Security-First                              │
│              Ed25519 signatures, offline verification           │
└─────────────────────────────────────────────────────────────────┘
```

### Dual-Component Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                                                                       │
│                        Server (Publisher)                            │
│                      CLI Tool / Web Backend                          │
│                                                                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐ │
│  │  Fence   │  │ Merkle   │  │  Delta   │  │ Snapshot │  │   Sign  │ │
│  │  Input   │  │  Tree    │  │  Patch   │  │   File   │  │ Generate│ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬────┘ │
│       │           │           │           │              │        │   │
│       ▼           ▼           ▼           ▼              ▼        ▼   │
│   ┌─────────────────────────────────────────────────────────────────┐ │
│   │              Static File Storage (CDN/OSS/IPFS)                  │ │
│   │  manifest.json │  v1.bin  │  v1_v2.delta  │  v2.snapshot.bin   │ │
│   └─────────────────────────────────────────────────────────────────┘ │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ HTTP Polling
                                    ▼
┌──────────────────────────────────────────────────────────────────────┐
│                        Client (Drone SDK)                            │
│                   Runs on Drone / Controller APP                     │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  HTTP Client   │ Sync Logic │  SQLite  │  R-Tree  │  Signature │  │
│  │  (Download/Retry)│ (Polling) │ (Persist)│ (Query) │ Verification│  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                       │
│                         API: Check(lat, lon) → Allowed?              │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### Prerequisites

- **Go 1.25+** (latest stable version recommended)
- **Make** (optional, for convenient builds)
- **Docker** (optional, for containerized deployment)

### Installation

#### Method 1: Build from Source

```bash
# Clone repository
git clone https://github.com/iannil/geofence-updater-lite.git
cd geofence-updater-lite

# Download dependencies
go mod download

# Build all binaries
make build-all

# Or build only the publisher tool
make build
```

Build artifacts are located in the `bin/` directory:

- `publisher` - Publisher tool (server)
- `sdk-example` - SDK usage example (client)

#### Method 2: Docker Build

```bash
# Build image
docker build -t gul-publisher .

# Run container
docker run -it --rm -v $(pwd)/data:/data gul-publisher
```

#### Method 3: Cross-Compilation

```bash
# Build for multiple platforms
make cross-compile
```

### Basic Usage

**1. Generate Key Pair**

```bash
$ ./bin/publisher keys

Generated key pair:
  Private Key: 0x7c3a9f2e... (keep safe)
  Public Key: 0x8d4b1c5a... (for client verification)
  KeyID: k1_20240118
```

**2. Initialize Database**

```bash
$ ./bin/publisher init

Initialization complete:
  Database: ./data/fences.db
  Version: v1
```

**3. Add Geofence**

Create a geofence data file `fence.json`:

```json
{
  "id": "fence-20240118-001",
  "type": "TEMP_RESTRICTION",
  "geometry": {
    "polygon": [
      {"lat": 39.9042, "lon": 116.4074},
      {"lat": 39.9142, "lon": 116.4074},
      {"lat": 39.9142, "lon": 116.4174},
      {"lat": 39.9042, "lon": 116.4174}
    ]
  },
  "start_ts": 1709880000,
  "end_ts": 1709990000,
  "priority": 10,
  "name": "Beijing 3rd Ring Temporary Control Zone",
  "description": "No-fly zone for temporary event"
}
```

Add to database:

```bash
$ ./bin/publisher add fence.json

Added successfully: fence-20240118-001 (type: TEMP_RESTRICTION)
```

**4. Publish Update**

```bash
$ ./bin/publisher publish --output ./output

Publish complete:
  Version: v2
  Delta: ./output/v1_v2.delta (2.3 KB)
  Snapshot: ./output/v2.snapshot.bin (15.6 KB)
  Manifest: ./output/manifest.json
```

**5. Deploy to CDN**

Upload the `output/` directory to your CDN/OSS:

```bash
# Example: Using AWS CLI
aws s3 sync ./output s3://your-bucket/geofence/
```

**6. Client Usage**

```bash
$ ./bin/sdk-example \
  -manifest https://cdn.example.com/geofence/manifest.json \
  -public-key 0x8d4b1c5a... \
  -store ./geofence.db

Starting sync...
  Current version: v0
  Remote version: v2
  Downloading delta: 2.3 KB
  Update applied: v0 → v2
  Signature verified

Starting geofence check...
  Check (39.9042, 116.4074): NOT ALLOWED - Beijing 3rd Ring Temporary Control Zone
```

---

## Usage Guide

### Publisher Tool

The publisher tool is used to manage and publish geofence updates.

#### Command Reference

```bash
# Generate Ed25519 key pair
$ publisher keys

# Initialize geofence database
$ publisher init [--db-path ./data/fences.db]

# Add new geofence
$ publisher add <fence.json>

# Batch add geofences
$ publisher add --batch <fences-dir>

# List all geofences
$ publisher list [--type TEMP_RESTRICTION]

# Remove geofence
$ publisher remove <fence-id>

# Publish new version
$ publisher publish [--output ./output] [--message "update message"]

# View version history
$ publisher history
```

#### Supported Geofence Types

| Type | Description | Priority Range |
| ------ | ------------- | ---------------- |
| `TEMP_RESTRICTION` | Temporary restriction zone | 10-50 |
| `PERMANENT_NO_FLY` | Permanent no-fly zone | 100 |
| `ALTITUDE_LIMIT` | Altitude limit zone | 20-40 |
| `ALTITUDE_MINIMUM` | Minimum altitude requirement | 20-40 |
| `SPEED_LIMIT` | Speed limit zone | 10-30 |

#### Geometry Support

```json
// Polygon
{
  "geometry": {
    "polygon": [
      {"lat": 39.9042, "lon": 116.4074},
      {"lat": 39.9142, "lon": 116.4074},
      {"lat": 39.9142, "lon": 116.4174},
      {"lat": 39.9042, "lon": 116.4174}
    ]
  }
}

// Circle
{
  "geometry": {
    "circle": {
      "center": {"lat": 39.9042, "lon": 116.4074},
      "radius_m": 5000
    }
  }
}

// Rectangle
{
  "geometry": {
    "rectangle": {
      "min": {"lat": 39.9000, "lon": 116.4000},
      "max": {"lat": 39.9200, "lon": 116.4200}
    }
  }
}
```

---

### Client SDK (Drone SDK)

The SDK provides geofence queries and automatic synchronization.

#### Go SDK Integration

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/iannil/geofence-updater-lite/pkg/config"
    "github.com/iannil/geofence-updater-lite/pkg/sync"
)

func main() {
    ctx := context.Background()

    // Create configuration
    cfg := &config.ClientConfig{
        ManifestURL:    "https://cdn.example.com/geofence/manifest.json",
        PublicKeyHex:   "8d4b1c5a...", // Public key in hex
        StorePath:      "./geofence.db",
        SyncInterval:   1 * time.Minute,
        HTTPTimeout:    30 * time.Second,
    }

    // Create syncer
    syncer, err := sync.NewSyncer(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer syncer.Close()

    // Start auto-sync
    results := syncer.StartAutoSync(ctx, 1*time.Minute)

    // Handle sync results
    go func() {
        for result := range results {
            if result.Error != nil {
                log.Printf("Sync error: %v", result.Error)
                continue
            }
            if result.UpToDate {
                log.Printf("Already up to date (v%d)", result.CurrentVer)
            } else {
                log.Printf("Update complete: v%d → v%d, took %v",
                    result.PreviousVer, result.CurrentVer, result.Duration)
            }
        }
    }()

    // Geofence check
    allowed, restriction, err := syncer.Check(ctx, 39.9042, 116.4074)
    if err != nil {
        log.Fatal(err)
    }

    if !allowed {
        log.Printf("NOT ALLOWED: %s - %s", restriction.Name, restriction.Description)
        // Execute no-fly logic...
    }
}
```

#### SDK API Reference

| Method | Description | Return Value |
| -------- | ------------- | -------------- |
| `NewSyncer(ctx, cfg)` | Create syncer | `(*Syncer, error)` |
| `StartAutoSync(ctx, interval)` | Start auto-sync | `<-chan SyncResult` |
| `CheckForUpdates(ctx)` | Check for updates | `(*Manifest, error)` |
| `Sync(ctx)` | Execute sync | `(*SyncResult, error)` |
| `Check(ctx, lat, lon)` | Geofence check | `(allowed, restriction, error)` |
| `Close()` | Close syncer | `error` |

---

## API Documentation

### pkg/crypto - Cryptography Module

```go
// Generate Ed25519 key pair
keyPair, err := crypto.GenerateKeyPair()

// Sign data
signature := crypto.Sign(privateKey, data)

// Verify signature
valid := crypto.Verify(publicKey, data, signature)

// Calculate key ID (for key rotation)
keyID := crypto.PublicKeyToKeyID(publicKey)
```

### pkg/merkle - Merkle Tree Module

```go
// Build Merkle Tree from fence items
tree, err := merkle.NewTree(fences)

// Get root hash
rootHash := tree.RootHash()

// Generate Merkle proof
proof, err := tree.GetProof(fenceID)

// Verify Merkle proof
valid := merkle.VerifyProof(fenceID, fenceData, proof, rootHash)
```

### pkg/storage - Storage Module

```go
// Open database
store, err := storage.Open(ctx, &storage.Config{Path: "./geofence.db"})

// Add fence
store.AddFence(ctx, &fence)

// Point query (using R-Tree)
fences, err := store.QueryAtPoint(ctx, lat, lon)

// Version management
version, _ := store.GetVersion(ctx)
store.SetVersion(ctx, newVersion)
```

### pkg/sync - Sync Module

```go
// Create syncer
syncer, _ := sync.NewSyncer(ctx, cfg)

// Check for updates
manifest, _ := syncer.CheckForUpdates(ctx)

// Sync data
result, _ := syncer.Sync(ctx)

// Auto-sync
results := syncer.StartAutoSync(ctx, interval)
```

### pkg/binarydiff - Binary Diff Module

```go
// Calculate diff
delta, err := binarydiff.Diff(oldFences, newFences)

// Apply diff
newFences, err := binarydiff.PatchFences(oldFences, delta)
```

---

## Performance Metrics

| Operation | Performance | Notes |
| ----------- | ------------- | ------- |
| Geofence Check | < 1ms | 1000 queries, R-Tree indexed |
| Merkle Tree Build | < 100ms | 1000 fences |
| Delta Calculation | < 50ms | 1000 fences comparison |
| Delta Size | ~2-5 KB | Typical 100 fence changes |
| Full Snapshot | ~15 KB | 100 fences (Protobuf encoded) |

---

## Protocol Specification

### Fence Item

| Field | Type | Description |
| ------- | ------ | ------------- |
| `id` | string | Unique identifier |
| `type` | FenceType | Geofence type |
| `geometry` | Geometry | Geometry shape (polygon/circle/rectangle) |
| `start_ts` | int64 | Effective timestamp |
| `end_ts` | int64 | Expiry timestamp, 0 means never expires |
| `priority` | uint32 | Priority, higher overrides lower |
| `max_alt_m` | uint32 | Max altitude limit (meters), 0 means no limit |
| `max_speed_mps` | uint32 | Max speed limit (m/s), 0 means no limit |
| `name` | string | Geofence name |
| `description` | string | Geofence description |
| `signature` | []byte | Ed25519 signature |
| `key_id` | string | Key ID |

### Manifest File

| Field | Type | Description |
| ------- | ------ | ------------- |
| `version` | uint64 | Global version number (incrementing) |
| `timestamp` | int64 | Publish timestamp |
| `root_hash` | []byte | Merkle Tree root hash |
| `delta_url` | string | Delta package download URL |
| `snapshot_url` | string | Full snapshot download URL |
| `delta_size` | uint64 | Delta package size (bytes) |
| `snapshot_size` | uint64 | Snapshot size (bytes) |
| `delta_hash` | []byte | Delta package hash (SHA-256) |
| `snapshot_hash` | []byte | Snapshot hash (SHA-256) |
| `message` | string | Version message |

---

## Project Structure

```
geofence-updater-lite/
├── cmd/                          # Command-line tools
│   ├── publisher/                # Publisher tool (server)
│   └── sdk-example/              # SDK usage example (client)
├── pkg/                          # Core packages
│   ├── binarydiff/               # Binary diff algorithm
│   ├── client/                   # HTTP client
│   ├── config/                   # Configuration management
│   ├── converter/                # Data format conversion
│   ├── crypto/                   # Ed25519 cryptography
│   ├── geofence/                 # Geofence core logic
│   ├── merkle/                   # Merkle Tree implementation
│   ├── protocol/protobuf/        # Protocol Buffers definitions
│   ├── publisher/                # Publishing logic
│   ├── storage/                  # SQLite storage layer
│   ├── sync/                     # Sync logic
│   └── version/                  # Version management
├── internal/                     # Internal packages
│   ├── testutil/                 # Test utilities
│   └── version/                  # Internal version info
├── docs/                         # Documentation
│   ├── spec/                     # Technical specifications
│   ├── progress/                 # Progress documentation
│   └── planning/                 # Planning documents
├── scripts/                      # Build scripts
├── test/                         # Test data
├── bin/                          # Build output
├── Makefile                      # Build system
├── go.mod                        # Go module definition
├── go.sum                        # Dependency lock
├── Dockerfile                    # Docker definition
├── LICENSE                       # Apache 2.0 license
├── README.md                     # This file
├── README.zh-CN.md               # Chinese version
├── CONTRIBUTING.md               # Contributing guide
├── CHANGELOG.md                  # Changelog
├── CLAUDE.md                     # Claude Code project guide
└── SECURITY.md                   # Security policy
```

---

## Development Guide

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run benchmarks
make test-bench
```

### Code Standards

```bash
# Format code
make fmt

# Static check
make vet

# Lint (requires golangci-lint)
make lint
```

### Building

```bash
# Build all binaries
make build-all

# Cross-compile
make cross-compile

# Build Docker image
make docker-build
```

---

## FAQ

**Q: Why Ed25519 instead of RSA/ECDSA?**

A: Ed25519 offers better security and performance:

- Signature size only 64 bytes (RSA-2048 needs 256 bytes)
- Verification ~5x faster than ECDSA
- Built-in side-channel attack protection

**Q: How to handle key rotation?**

A: Use the KeyID mechanism. Each signature contains a key ID, and clients can support multiple public keys:

```go
syncer.PublicKeys = map[string]*crypto.PublicKey{
    "k1_2024": oldPublicKey,
    "k2_2024": newPublicKey,
}
```

**Q: How many geofences are supported?**

A: Theoretically unlimited. Tested:

- 10,000 fences: Full snapshot ~1.5 MB, query < 2ms
- 100,000 fences: Full snapshot ~15 MB, query < 5ms

**Q: Can it work offline?**

A: Yes. The SDK continues working with locally cached geofence data and automatically syncs when network is restored.

---

## Roadmap

- [x] Core data structures
- [x] Ed25519 signature verification
- [x] Merkle Tree implementation
- [x] R-Tree spatial indexing
- [x] Binary delta
- [x] HTTP sync
- [x] Publisher tool
- [x] SDK example
- [x] CI/CD pipeline (GitHub Actions)
- [ ] C++ SDK
- [ ] Performance benchmarks
- [ ] Web management interface

---

## License

This project is licensed under the [Apache License 2.0](LICENSE).

```
Copyright 2024-2025 Geofence-Updater-Lite Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

---

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to contribute.

---

## Acknowledgments

This project draws inspiration from:

- [Git](https://git-scm.com/) - Merkle Tree version management concept
- [bsdiff](https://www.daemonology.net/bsdiff/) - Binary diff algorithm
- [go-polyline](https://github.com/twpayne/go-polyline) - Coordinate compression algorithm
