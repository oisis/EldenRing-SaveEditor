VERSION := 2.0.0

GO_PACKAGES := \
	./backend/apperror \
	./backend/gamecatalog/... \
	./backend/hostsettings \
	./backend/deployment \
	./backend/safetyprofile \
	./backend/saveengine/... \
	./backend/buildtemplates/... \
	./backend/endpoints/... \
	./internal/catalogassets \
	./internal/desktop \
	./tools/...

# Wails CLI pinned to the version the frontend baseline was verified against.
WAILS := go run github.com/wailsapp/wails/v2/cmd/wails@v2.15.0

# Desktop build target. Supported values: darwin/arm64, windows/amd64,
# linux/amd64. Selecting a platform does not by itself provide a working
# cross-compilation toolchain: each target needs its own CGO toolchain and
# native GUI dependencies, so a foreign target is normally built on its own
# runner.
PLATFORM ?= darwin/arm64

# The Linux dependency set is built against WebKit2GTK 4.1, which Wails only
# selects through this build tag. The tag is Linux-only and must not leak into
# the macOS or Windows builds.
WAILS_BUILD_TAGS := $(if $(filter linux/%,$(PLATFORM)),-tags webkit2_41,)

.PHONY: all deps test test-race vet bindings frontend-check frontend-test frontend-build app-build swagger-start swagger-stop swagger-restart viewer-start viewer-stop viewer-restart help

all: test

deps:
	go mod download

test:
	go test -count=1 $(GO_PACKAGES)

test-race:
	go test -race -count=1 $(GO_PACKAGES)

vet:
	go vet $(GO_PACKAGES)

bindings:
	$(WAILS) generate module

frontend-check:
	pnpm --dir frontend run check
	pnpm --dir frontend run typecheck

frontend-test:
	pnpm --dir frontend run test

frontend-build:
	pnpm --dir frontend run build

# VERSION stays the single source of the release version: it reaches the
# application only through the linker, never through a Go or TypeScript
# constant.
#
# -m skips the CLI's `go mod tidy` and -nosyncgomod its go.mod rewrite, for the
# same reason `test` does not use `./...`: the ignored scratch programs under
# tmp/ are not part of the application and must not take part in resolving the
# module graph.
app-build:
	$(WAILS) build -platform $(PLATFORM) $(WAILS_BUILD_TAGS) -clean -m -nosyncgomod \
		-ldflags "-X main.applicationVersion=$(VERSION)"

swagger-start:
	./tools/run_swagger.sh start -app-version "$(VERSION)"

swagger-stop:
	./tools/run_swagger.sh stop

swagger-restart:
	./tools/run_swagger.sh restart -app-version "$(VERSION)"

viewer-start:
	./tools/run_viewer.sh start

viewer-stop:
	./tools/run_viewer.sh stop

viewer-restart:
	./tools/run_viewer.sh restart

help:
	@echo "SaveForge 2.0 development commands:"
	@echo "  make deps              Download Go dependencies"
	@echo "  make test              Run the SaveForge 2.0 test suite"
	@echo "  make test-race         Run the SaveForge 2.0 test suite with the race detector"
	@echo "  make vet               Run go vet for SaveForge 2.0 packages"
	@echo "  make bindings          Generate the Wails bindings in frontend/wailsjs"
	@echo "  make frontend-check    Run Biome and the TypeScript typecheck"
	@echo "  make frontend-test     Run the frontend Vitest suite"
	@echo "  make frontend-build    Build the production frontend bundle"
	@echo "  make app-build         Build the desktop app with VERSION (PLATFORM=darwin/arm64)"
	@echo "                         make app-build PLATFORM=windows/amd64"
	@echo "                         make app-build PLATFORM=linux/amd64"
	@echo "  make swagger-start     Start Scalar Docs and the local API host"
	@echo "  make swagger-stop      Stop Scalar Docs and the local API host"
	@echo "  make swagger-restart   Restart Scalar Docs and the local API host"
	@echo "  make viewer-start      Start the GameCatalog DB Viewer"
	@echo "  make viewer-stop       Stop the GameCatalog DB Viewer"
	@echo "  make viewer-restart    Restart the GameCatalog DB Viewer"
