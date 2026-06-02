package cli

import "strings"

const fallbackVersion = "0.0.0-dev"

var (
	version = fallbackVersion
	commit  = ""
	date    = ""
)

func Version() string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return fallbackVersion
	}
	return strings.TrimPrefix(trimmed, "v")
}

func VersionText() string {
	return "mojify " + Version() + "\n"
}
