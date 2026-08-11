VERSION := 2.0.0

GO_PACKAGES := \
	./backend/gamecatalog/... \
	./backend/saveengine/... \
	./backend/endpoints/... \
	./tools/...

.PHONY: all deps test test-race vet swagger-start swagger-stop swagger-restart viewer-start viewer-stop viewer-restart help

all: test

deps:
	go mod download

test:
	go test -count=1 $(GO_PACKAGES)

test-race:
	go test -race -count=1 $(GO_PACKAGES)

vet:
	go vet $(GO_PACKAGES)

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
	@echo "  make swagger-start     Start Scalar Docs and the local API host"
	@echo "  make swagger-stop      Stop Scalar Docs and the local API host"
	@echo "  make swagger-restart   Restart Scalar Docs and the local API host"
	@echo "  make viewer-start      Start the GameCatalog DB Viewer"
	@echo "  make viewer-stop       Stop the GameCatalog DB Viewer"
	@echo "  make viewer-restart    Restart the GameCatalog DB Viewer"
