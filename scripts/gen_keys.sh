#!/bin/bash
# gen_keys.sh generates Ed25519 key pairs for GUL

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default output directory
OUTPUT_DIR="${OUTPUT_DIR:-$PROJECT_ROOT/keys}"
mkdir -p "$OUTPUT_DIR"

echo "Generating Ed25519 key pair..."
echo "Output directory: $OUTPUT_DIR"

# Use the publisher tool to generate keys
cd "$PROJECT_ROOT"
go run ./cmd/publisher keys 2>&1 | tee "$OUTPUT_DIR/key_info.txt"

# Extract keys to separate files
if [ -f "$OUTPUT_DIR/key_info.txt" ]; then
    grep "Public Key:" "$OUTPUT_DIR/key_info.txt" | awk '{print $3}' > "$OUTPUT_DIR/public.key"
    grep "Private Key:" "$OUTPUT_DIR/key_info.txt" | awk '{print $3}' > "$OUTPUT_DIR/private.key"
    grep "Key ID:" "$OUTPUT_DIR/key_info.txt" | awk '{print $3}' > "$OUTPUT_DIR/key_id.txt"

    echo ""
    echo "Key files created:"
    echo "  - $OUTPUT_DIR/public.key"
    echo "  - $OUTPUT_DIR/private.key"
    echo "  - $OUTPUT_DIR/key_id.txt"
    echo "  - $OUTPUT_DIR/key_info.txt"

    echo ""
    echo "IMPORTANT: Keep the private key secure and never commit it to version control!"
fi
