package main

import (
	"log/slog"
	"os"
	"pkgmate/ui"
)

func main() {
	isPrivileged := os.Geteuid() == 0
	if isPrivileged {
		slog.Error("can't run in root")
		os.Exit(1)
	}
	err := ui.Run()
	if err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
