BINARY  := call-recorder
DIST    := dist
LDFLAGS := -ldflags="-s -w"

.PHONY: build build-cli install clean dist tag

# Full build (with tray)
build:
	go build $(LDFLAGS) -o $(BINARY) .

# CLI-only build (no tray, no libappindicator dependency)
build-cli:
	go build $(LDFLAGS) -tags notray -o $(BINARY)-cli .

INSTALL_DIR ?= $(shell dirname $(shell which $(BINARY) 2>/dev/null || echo /usr/local/bin/$(BINARY)))

install:
	go build $(LDFLAGS) -o $(BINARY) .
	sudo install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)

clean:
	rm -f $(BINARY) $(BINARY)-cli
	rm -rf $(DIST)

# ── Local cross-compilation ───────────────────────────────────────────────────

dist: dist-linux-amd64 dist-linux-arm64 dist-windows dist-macos-note

dist-linux-amd64:
	@mkdir -p $(DIST)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-amd64 .
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build $(LDFLAGS) -tags notray -o $(DIST)/$(BINARY)-linux-amd64-cli .
	@echo "✓ linux/amd64"

dist-linux-arm64:
	@mkdir -p $(DIST)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
		go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-arm64 .
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
		go build $(LDFLAGS) -tags notray -o $(DIST)/$(BINARY)-linux-arm64-cli .
	@echo "✓ linux/arm64"

dist-windows:
	@mkdir -p $(DIST)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
		go build $(LDFLAGS) -o $(DIST)/$(BINARY)-windows-amd64.exe .
	@echo "✓ windows/amd64"

dist-macos-note:
	@echo ""
	@echo "⚠  macOS: compila direttamente su Mac oppure usa: make tag"
	@echo ""

# ── Release ───────────────────────────────────────────────────────────────────

tag:
	@test -n "$(V)" || (echo "Usa: make tag V=1.2.3"; exit 1)
	git tag v$(V)
	git push gitlab v$(V)
