#!/bin/sh
# Suite quality gate for gare (Go family)
# Enforces: go vet ./..., go test ./..., runs filet check

set -euo pipefail

# Run go vet for static analysis
echo "Running go vet..."
go vet ./...

# Run tests
echo "Running tests..."
go test ./...

# Run filet check for code quality
echo "Running filet check..."
filet check .

echo "All checks passed"