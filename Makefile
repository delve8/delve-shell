.PHONY: build clean

BINARY := bin/delve-shell

build:
	@mkdir -p bin
	go build -o $(BINARY) ./cmd/delve-shell

clean:
	rm -rf bin/
