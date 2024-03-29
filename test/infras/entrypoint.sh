#!/bin/sh

# Terminate script immediately on any errors
set -e

# Define Go versions
go_versions=${GO_VERSIONS}

echo ${GO_VERSIONS}

# Iterate over each version
for version in $go_versions; do
    # Set Go version using gobrew
    gobrew use "$version"
    
    # Execute Test as defined in the Makefile
    make go-test
done