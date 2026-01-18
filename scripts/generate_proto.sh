#!/bin/bash
# generate_proto.sh generates Go code from Protobuf definitions

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$PROJECT_ROOT/pkg/protocol/protobuf"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc not found. Please install Protocol Buffers compiler."
    echo "  macOS: brew install protobuf"
    echo "  Ubuntu: apt-get install protobuf-compiler"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Error: protoc-gen-go not found. Installing..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

echo "Generating Go code from Protobuf definitions..."

# Generate Go code from proto files
protoc \
    --proto_path="$PROTO_DIR" \
    --go_out="$PROTO_DIR" \
    --go_opt=paths=source_relative \
    "$PROTO_DIR/fence.proto" \
    "$PROTO_DIR/manifest.proto"

echo "Protobuf code generation complete!"
echo "Generated files:"
echo "  - fence.pb.go"
echo "  - manifest.pb.go"
