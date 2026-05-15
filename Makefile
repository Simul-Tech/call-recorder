BINARY  := call-recorder
DIST    := dist

.PHONY: build install clean dist

build:
	go build -o $(BINARY) .

install:
	go install .
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)
	rm -rf $(DIST)

dist: dist-linux-amd64 dist-linux-arm64 dist-windows dist-macos-note

dist-linux-amd64:
	@mkdir -p $(DIST)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build -ldflags="-s -w" -o $(DIST)/$(BINARY)-linux-amd64 .
	@echo "✓ linux/amd64"

dist-linux-arm64:
	@mkdir -p $(DIST)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
		go build -ldflags="-s -w" -o $(DIST)/$(BINARY)-linux-arm64 .
	@echo "✓ linux/arm64"

dist-windows:
	@mkdir -p $(DIST)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
		go build -ldflags="-s -w" -o $(DIST)/$(BINARY)-windows-amd64.exe .
	@echo "✓ windows/amd64"

dist-macos-note:
	@echo ""
	@echo "⚠  macOS: i binari per darwin vanno compilati su una macchina macOS."
	@echo "   Usa GitHub Actions (make tag) oppure compila direttamente su Mac."
	@echo ""

tag:
	@test -n "$(V)" || (echo "Usa: make tag V=1.2.3"; exit 1)
	git tag v$(V)
	git push gitlab v$(V)
