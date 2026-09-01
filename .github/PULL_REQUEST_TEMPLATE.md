## What this changes

<!-- One or two sentences. Link the issue if there is one. -->

## Checks

- [ ] `gofmt -l .` prints nothing
- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes
- [ ] No cookie dump, token or session file in the diff

## If this changes a request to Icons8

- [ ] `docs/api.md` updated
- [ ] A smoke check covers it, and would fail if the parameter were ignored

Several Icons8 parameters return HTTP 200 with unfiltered results when the name
is wrong, so a check that only asserts the call succeeded proves nothing.
