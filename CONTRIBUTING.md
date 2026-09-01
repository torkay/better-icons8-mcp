# Contributing

Bug reports and patches are welcome.

## Before opening a PR

```sh
gofmt -l .      # must print nothing
go vet ./...
go test ./...   # offline, no session needed
go build ./...
```

CI runs these four. If the change touches anything that talks to Icons8, run the
live suite as well. It needs a working session.

```sh
go run ./cmd/smoke -bin ./dist/icons8-mcp
```

## Changing a request

Read `docs/api.md` first. Several Icons8 parameters fail silently. A wrong
parameter name returns HTTP 200 with unfiltered results, so a broken query looks
like a working one.

Prove a query change with a smoke check that would fail if the filter were
ignored. Compare a filtered count against an unfiltered one rather than
asserting the call succeeded.

`cmd/recon` and `cmd/reconflow` exist for when the API moves and you need to see
what the web app sends.

## Never commit

Cookie dumps, `session.json`, tokens, or anything else carrying a real session.
`.gitignore` covers the obvious paths. Check `git diff --staged` anyway.

## Tests

Unit tests run offline. Anything needing the live API belongs in `cmd/smoke`,
which is a program rather than a test package for that reason.
