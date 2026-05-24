.PHONY: build test tidy clean run-index run-search lint fmt install uninstall release release-snapshot coverage

BIN := bin/chriscast
PKG := ./cmd/chriscast
TAGS ?=

build:
	CGO_ENABLED=1 go build -tags "$(TAGS)" -o $(BIN) $(PKG)

build-ui:
	$(MAKE) build TAGS=gtk

tidy:
	go mod tidy

test:
	go test ./...

clean:
	rm -rf bin

run-index: build
	$(BIN) index

run-search: build
	$(BIN) search "$(Q)"

lint:
	golangci-lint run ./...

fmt:
	gofumpt -w .

install: build-ui
	mkdir -p $(HOME)/.local/bin
	mkdir -p $(HOME)/.config/systemd/user
	mkdir -p $(HOME)/.local/share/applications
	mkdir -p $(HOME)/.local/share/icons/hicolor/scalable/apps
	cp $(BIN) $(HOME)/.local/bin/chriscast
	cp dist/chriscast.service $(HOME)/.config/systemd/user/chriscast.service
	cp dist/chriscast.desktop $(HOME)/.local/share/applications/chriscast.desktop
	cp dist/chriscast.svg $(HOME)/.local/share/icons/hicolor/scalable/apps/chriscast.svg
	-update-desktop-database $(HOME)/.local/share/applications 2>/dev/null || true
	-gtk-update-icon-cache -f -t $(HOME)/.local/share/icons/hicolor 2>/dev/null || true
	systemctl --user daemon-reload
	systemctl --user enable chriscast.service
	systemctl --user start chriscast.service
	@echo ""
	@echo "Installed. chriscast now appears in your app menu."
	@echo "Daemon running. Press Ctrl+Space to summon overlay."

uninstall:
	-systemctl --user stop chriscast.service
	-systemctl --user disable chriscast.service
	rm -f $(HOME)/.config/systemd/user/chriscast.service
	rm -f $(HOME)/.local/share/applications/chriscast.desktop
	rm -f $(HOME)/.local/share/icons/hicolor/scalable/apps/chriscast.svg
	-update-desktop-database $(HOME)/.local/share/applications 2>/dev/null || true
	-gtk-update-icon-cache -f -t $(HOME)/.local/share/icons/hicolor 2>/dev/null || true
	systemctl --user daemon-reload
	rm -f $(HOME)/.local/bin/chriscast

release:
	goreleaser release --clean

release-snapshot:
	goreleaser release --snapshot --clean

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
