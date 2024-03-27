# Define Go versions
$go_versions = $env:GO_VERSIONS -split ','

foreach ($version in $go_versions) {
    gobrew install $version
}
