package main

import (
	"fmt"

	"github.com/Jguer/go-alpm/v2"
)

func main() {
	h, err := alpm.Initialize("/", "/var/lib/pacman/")
	if err != nil {
		fmt.Printf("Failed to initialize: %v\n", err)
		return
	}
	defer h.Release()

	db, _ := h.LocalDB()
	pkgs := db.PkgCache()

	pkgs.ForEach(func(pkg alpm.IPackage) error {
		fmt.Printf("%s %s\n", pkg.Name(), pkg.Version())
		return nil
	})

	fmt.Printf("\nTotal packages: %d\n", len(pkgs.Slice()))
}
