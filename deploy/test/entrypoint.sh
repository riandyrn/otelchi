#!/bin/sh

# Terminate script immediately on any errors
set -e

# Define Go versions
go_versions="1.16 1.17 1.18 1.19 1.20"

# Iterate over each version
for version in $go_versions; do
    # Set Go version using gobrew
    gobrew use "$version"
    
    # Build and test
    go build .
    go test ./...
done