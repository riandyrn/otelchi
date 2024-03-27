# Define Go versions
$go_versions = $env:GO_VERSIONS -split ' '

Write-Host $env:GO_VERSIONS

foreach ($version in $go_versions) {
    # Set Go version using gobrew
    gobrew use $version

    # Build and test
    go build .
    go test ./...
}
