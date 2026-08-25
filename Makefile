BINARY := toudocu
CMD := ./cmd/toudocu
DIST := dist
INSTALL_DIR ?= $(HOME)/.local/bin
TOUDOCU := go run $(CMD)
DOCS_DIR := docs

.PHONY: fmt fmt-check lint test web web-check browser-test check build update-local docs docs-serve landing-serve clean release

fmt:
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l .)"

lint:
	golangci-lint run ./...

test:
	go test ./...
	go test -race ./...
	npm --prefix web run typecheck
	npm --prefix web test

web:
	npm --prefix web run build

web-check:
	npm --prefix web run typecheck
	npm --prefix web test
	npm --prefix web run build
	git diff --exit-code -- internal/site/assets/generated

browser-test:
	npm --prefix web run test:browser

check: fmt-check test web-check
	go mod verify
	$(TOUDOCU) check ./$(DOCS_DIR) --repository-root . --strict --stale-days 0

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) $(CMD)

update-local: build
	install -Dm755 "$(BINARY)" "$(INSTALL_DIR)/$(BINARY)"

docs:
	$(TOUDOCU) build ./$(DOCS_DIR) --output ./build/project-docs --repository-root . --clean
	$(TOUDOCU) build ./docs-en --output ./build/project-docs/en --repository-root . --clean

docs-serve:
	$(TOUDOCU) serve ./$(DOCS_DIR)

landing-serve:
	node landing/dev-server.mjs

release: check
	rm -rf $(DIST)
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(DIST)/toudocu-linux-amd64 $(CMD)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(DIST)/toudocu-linux-arm64 $(CMD)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(DIST)/toudocu-darwin-amd64 $(CMD)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(DIST)/toudocu-darwin-arm64 $(CMD)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(DIST)/toudocu-windows-amd64.exe $(CMD)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(DIST)/toudocu-windows-arm64.exe $(CMD)
	cp LICENSE $(DIST)/
	{ cat THIRD_PARTY_NOTICES.md; printf '\n\n# Bundled browser dependency licenses (JSON)\n'; cat internal/site/assets/generated/licenses.json; printf '\n\n# Vendored browser asset notices\n'; cat internal/site/assets/generated/mermaid.LICENSE.txt; printf '\n\n'; cat internal/site/assets/generated/codemirror.LICENSE.txt; printf '\n\n'; cat internal/site/assets/generated/swagger-ui.LICENSE.txt; printf '\n\n'; cat internal/site/assets/generated/swagger-ui-bundle.LICENSE.txt; printf '\n\n'; cat internal/site/assets/generated/swagger-ui-standalone-preset.LICENSE.txt; } > $(DIST)/THIRD_PARTY_NOTICES.md
	cp internal/site/assets/generated/codemirror.checksums.txt $(DIST)/CODEMIRROR-CHECKSUMS.txt
	cp internal/site/assets/generated/swagger-ui.checksums.txt $(DIST)/SWAGGER-UI-CHECKSUMS.txt
	cp scripts/install.sh scripts/install.ps1 $(DIST)/
	cd $(DIST) && sha256sum * > checksums.txt

clean:
	rm -rf $(BINARY) $(DIST)
