BIN := dist/icons8-mcp

.PHONY: build check test smoke demo clean

build:
	go build -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/icons8-mcp

check:
	gofmt -l . | tee /dev/stderr | (! read)
	go vet ./...

test:
	go test ./...

# Needs a live Icons8 session; downloads real assets.
smoke: build
	go run ./cmd/smoke -bin $(BIN)

demo: build
	vhs demo/quickstart.tape
	vhs demo/tools.tape
	vhs demo/live.tape

clean:
	rm -rf dist
