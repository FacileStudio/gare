package main

// version is what a plain `go build` reports. GoReleaser overrides it at release time with
// `-X main.version={{.Version}}`, so this literal and the git tag move together in each release commit.
var version = "0.17.0"
