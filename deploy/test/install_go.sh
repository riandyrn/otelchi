#!/bin/sh

# Define Go versions
go_versions=${GO_VERSIONS}

# Install each Go version using gobrew
for version in $go_versions; do
    gobrew install "$version"
done