BIN := dist/icons8-mcp

.PHONY: build check test smoke plugin demo clean

build:
	go build -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/icons8-mcp

check:
	gofmt -l . | tee /dev/stderr | (! read)
	go vet ./...
	claude plugin validate .
	claude plugin validate ./plugins/icons8
	sh -n plugins/icons8/bin/icons8-mcp
	sh -n scripts/install.sh

test:
	go test ./...

# Needs a live Icons8 session; downloads real assets.
smoke: build
	go run ./cmd/smoke -bin $(BIN)

# Load the plugin without installing it from the marketplace.
plugin: build
	ICONS8_MCP_BIN=$(PWD)/$(BIN) claude --plugin-dir ./plugins/icons8

demo: build
	vhs demo/quickstart.tape
	vhs demo/usage.tape
	vhs demo/live.tape
	SPEED=1.6 bash demo/optimise.sh quickstart.gif
	# The Claude Code session is the long one, so it gets the smaller frame.
	SPEED=1.6 FPS=10 WIDTH=1000 COLORS=96 bash demo/optimise.sh usage.gif
	bash demo/optimise.sh live.gif
	bash demo/social-preview.sh

clean:
	rm -rf dist
