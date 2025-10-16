package main

import (
	"fmt"
	"os"
	"regexp"
)

func main() {
	version := os.Getenv("VERSION")
	if version == "" {
		fmt.Println("::error ::VERSION is required but missing")
		os.Exit(1)
	}

	if err := updatePKGBUILD(version); err != nil {
		fmt.Printf("::error ::Failed to update PKGBUILD: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("PKGBUILD updated successfully")
}

func updatePKGBUILD(version string) error {
	content, err := os.ReadFile("./template/PKGBUILD")
	if err != nil {
		return err
	}

	versionRegex := regexp.MustCompile(`pkgver=.*`)
	updated := versionRegex.ReplaceAllString(string(content), fmt.Sprintf("pkgver=%s", version))

	return os.WriteFile("./pkgbuild/PKGBUILD", []byte(updated), 0644)
}
