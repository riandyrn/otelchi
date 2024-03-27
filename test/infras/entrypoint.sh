#!/bin/sh

# Terminate script immediately on any errors
set -e

# Build and test
go build .
go test ./...