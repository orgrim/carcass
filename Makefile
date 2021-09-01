all: carcass

carcass: $(shell find . -name '*.go')
	go build -ldflags="-s -w" -o carcass ./cmd/carcass

install:
	go install -ldflags="-s -w" ./cmd/carcass

clean:
	-rm carcass

.PHONY: all clean
