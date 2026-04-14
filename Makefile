.PHONY: build build-all clean notices

BINARY := bin/delve-shell
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X delve-shell/internal/version.Version=$(VERSION) -X delve-shell/internal/version.Commit=$(COMMIT) -X delve-shell/internal/version.BuildDate=$(BUILD_DATE)
# 多平台：linux/darwin/windows × amd64/arm64
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/delve-shell

# build-all 交叉编译到 bin/delve-shell_<os>_<arch>[.exe]
build-all:
	@mkdir -p bin
	@for p in $(PLATFORMS); do \
		os=$$(echo $$p | cut -d'/' -f1); \
		arch=$$(echo $$p | cut -d'/' -f2); \
		out="bin/delve-shell_$${os}_$${arch}"; \
		[ "$$os" = "windows" ] && out="$$out.exe"; \
		echo "Building $$os/$$arch -> $$out"; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$out ./cmd/delve-shell || exit 1; \
	done
	@echo "Done. Artifacts in bin/"

clean:
	rm -rf bin/

notices:
	PLATFORMS="$(PLATFORMS)" ./scripts/update-third-party-notices.sh
