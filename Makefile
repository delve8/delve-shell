.PHONY: build build-all clean

BINARY := bin/delve-shell
# 多平台：linux/darwin/windows × amd64/arm64
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

build:
	@mkdir -p bin
	go build -o $(BINARY) ./cmd/delve-shell

# build-all 交叉编译到 bin/delve-shell_<os>_<arch>[.exe]
build-all:
	@mkdir -p bin
	@for p in $(PLATFORMS); do \
		os=$$(echo $$p | cut -d'/' -f1); \
		arch=$$(echo $$p | cut -d'/' -f2); \
		out="bin/delve-shell_$${os}_$${arch}"; \
		[ "$$os" = "windows" ] && out="$$out.exe"; \
		echo "Building $$os/$$arch -> $$out"; \
		GOOS=$$os GOARCH=$$arch go build -o $$out ./cmd/delve-shell || exit 1; \
	done
	@echo "Done. Artifacts in bin/"

clean:
	rm -rf bin/
