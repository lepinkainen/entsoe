# Repository Guidelines

## Project Structure & Module Organization
The Go module lives at the repository root, with `main.go` containing the CLI entrypoint and `entsoe.go` encapsulating ENTSO-E API logic. Supporting integration helpers and tests live alongside their corresponding files (e.g., `main_test.go`, `entsoe_test.go`). Build artefacts are written to `build/` and the default binary name is `entsoe_redis`; keep this directory out of source control except for reproducible assets. Shared documentation for LLM integrations sits under `llm-shared/`, and runtime configuration defaults ship in `config.yml` with optional overrides coming from `.env`.

## Build, Test, and Development Commands
- `task build`: run vet, lint, tests, and compile the local binary into `build/entsoe_redis`.
- `task build-linux`: cross-compile for Linux (`build/entsoe_redis-linux-amd64`) after passing quality gates.
- `task lint`: execute `golangci-lint run ./...`.
- `task test`, `task test-ci`: run `go test` suites; the CI variant enables the `ci` build tag and coverage.
- `go run .`: launch the CLI using the current workspace.
Use `task --list` for further automation helpers such as `upgrade-deps`.

## Coding Style & Naming Conventions
Adhere to Go 1.24 defaults: gofmt-formatted files with tab indentation and trailing newline. Exported symbols must carry GoDoc-style comments. Prefer descriptive package-level names (`entsoeClient`, `redisPool`) and keep filenames lowercase_with_underscores only for generated assets; otherwise stick to camelCase identifiers inside Go code. Lint failures from `golangci-lint` must be resolved before merging.

## Testing Guidelines
Place tests in the same directory as the code with the `_test.go` suffix. Use table-driven patterns to cover API edge cases and stub external services via in-memory Redis where possible. Run `task test` before submitting, and ensure CI passes `task test-ci` to maintain coverage parity. Name tests following `TestFunction_Scenario` for clarity.

## Commit & Pull Request Guidelines
Commits follow the existing short, imperative style (`Add shared llm instructions`). Keep the subject under ~60 characters, capitalized, and avoid trailing punctuation. Group related changes per commit. Pull requests should summarise intent, link to any tracked issue, and include test evidence (`task test` output or artefacts) plus configuration notes when changing `config.yml`. Mention breaking changes in a dedicated paragraph and request at least one reviewer familiar with the ENTSO-E integration.
