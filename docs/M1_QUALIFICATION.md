# M1 qualification

M1 qualification covers formatting, unit tests, static analysis, native Go build,
and OCI container build.

## Gates

- `gofmt -l ./cmd` returns no files
- `go test ./...` passes
- `go vet ./...` passes
- `go build ./cmd/number-station-tools` passes
- `docker build -f Containerfile .` passes
- observation persistence survives store reopen
- observations referencing an unknown station are rejected

The repository CI executes these gates with Go 1.25 on every pull request and
push to `main`.

## Local qualification note

The qualification runner used while introducing these gates had Go 1.23.2
available locally and no outbound access to download the Go 1.25 toolchain.
The source was therefore additionally checked for formatting and exercised with
`go test`, `go vet`, and `go build` under Go 1.23.2, while GitHub Actions is the
authoritative Go 1.25/container qualification environment.
