.PHONY: build arch debian brew clean help

help:
	@echo "Available targets:"
	@echo "  make build   - Build binaries with goreleaser"
	@echo "  make arch    - Run Arch Linux container"
	@echo "  make debian  - Run Debian container"
	@echo "  make brew    - Run Ubuntu/Brew container"
	@echo "  make clean   - Clean up containers and volumes"

build:
	goreleaser build --snapshot --clean

arch: build
	docker compose run --rm arch bash

debian: build
	docker compose run --rm debian bash

brew: build
	docker compose run --rm brew bash

clean:
	docker compose down -v
	docker compose rm -f
