# Release Process

This document explains how to create releases in this project in each release scenario.

Currently there are 2 release procedures for this project:

- [Latest Release](#latest-release)
- [Non-Latest Release](#non-latest-release)

## Latest Release

This procedure is used when we want to create new release from the latest development edge (latest commit in the `master` branch).

The steps for this procedure are the following:

1. Create new branch from the `master` branch with the following name format: `pre_release/v{MAJOR}.{MINOR}.{BUILD}`. For example, if we want to release for version `0.6.0`, we will first create a new branch from the `master` called `pre_release/v0.6.0`.
2. Update method `Version()` in `version.go` to return the target version.
3. Update the `CHANGELOG.md` to include all the notable changes.
4. Create a new PR from this branch to `master` with the title: `Release v{MAJOR}.{MINOR}.{BUILD}` (e.g `Release v0.6.0`).
5. One maintainers should at least approve the PR.
6. Upon approval, the PR will be merged to `master`.
7. Create new release with the description & tag matches the content inside `CHANGELOG.md`. Set the title to `Release v{MAJOR}.{MINOR}.{BUILD}` (e.g `Release v0.6.0`).
8. Set the release as the latest release.
9. Done.

## Non-Latest Release