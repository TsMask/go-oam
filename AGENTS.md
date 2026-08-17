# Repository Guidelines

## Project Structure & Module Organization

This is a Go module (`github.com/tsmask/go-oam`) targeting Go 1.25+. The public SDK entry point is `oam.go`. Feature modules live in `ws/` (WebSocket transport and codecs), `push/` (HTTP delivery), and `pkg/` (independent utilities such as files, sockets, SSH, and state collection). Tests are colocated as `*_test.go`. Runnable examples and static browser assets are under `examples/`; module-level documentation is in `README.md` and package READMEs.

## Build, Test, and Development Commands

```bash
go test ./...                      # Run the full test suite
go test -race ./pkg/file           # Check concurrent file operations
go build ./...                     # Compile all packages
go vet ./...                       # Run standard static analysis
gofmt -w .                         # Format all Go source
go run ./examples/ws/server        # Run a local WebSocket example
GOOS=linux GOARCH=amd64 go build ./...  # Cross-compile
```

## Coding Style & Naming Conventions

Follow standard Go formatting and idioms: tabs for indentation, `gofmt`-clean files, camelCase for unexported identifiers, PascalCase for exported identifiers, and meaningful context/error wrapping. Keep `pkg/` free of dependencies on `ws/` or `push/`. Do not hand-edit `ws/protocol/ws.pb.go`; update `ws/protocol/ws.proto` and regenerate it.

## Testing Guidelines

Use Go's built-in testing package. Place table-driven tests beside the code as `<unit>_test.go`, for example `pkg/socket/client_test.go`. Add tests for bug fixes and behavior changes, especially concurrency-sensitive paths; run `go test -race` for code using goroutines or shared state. Keep example changes runnable and covered by manual verification.

## Commit & Pull Request Guidelines

Recent history follows Conventional Commits, commonly `type(scope): summary` (for example, `docs(readme): ...`, `feat(telnet): ...`); keep subjects concise and use lowercase types. Pull requests should describe the change and motivation, list verification performed, mention breaking API changes, and link related issues. Include screenshots for UI changes in `examples/ws/web`.

## Security & Configuration Tips

Do not commit credentials, private keys, server addresses, or generated local artifacts. Keep security-sensitive examples parameterized through environment variables or local configuration, and preserve platform-specific behavior in SSH, PTY, and filesystem code.
