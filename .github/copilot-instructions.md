# Copilot instructions for edge-watcher

## Repository overview

This repository is a small Go CLI for monitoring Edge TPU devices on Linux. The entire application logic currently lives in `src/main.go`, and the Go module is rooted at `src/` rather than the repository root.

The program reads live device state directly from Linux sysfs paths under `/sys/class/apex`, enumerates only `apex_N` device directories, polls each device every second, and renders a table in the terminal using `pterm`.

## Build, test, and lint commands

Run commands from the `src/` directory because that is where `go.mod` lives.

- Build the project:
  - `cd src && go build ./...`
- Run the full Go test suite:
  - `cd src && go test ./...`
- Run a single test (when tests are added):
  - `cd src && go test ./... -run 'TestName'`
- Optional static check:
  - `cd src && go vet ./...`
- Format Go files before finishing a change:
  - `cd src && gofmt -w *.go`

The focused tests in `src/main_test.go` cover device discovery, sysfs parsing, error reporting, and value formatting. The practical verification path is `go test ./...` and `go build ./...` after changes.

## High-level architecture

- `src/main.go` is the main program.
- `getTpus(basePath string)` reads `/sys/class/apex` and collects only device directories whose names parse as `apex_0`, `apex_1`, etc.
- `readTpuStats` reads each device's own sysfs attributes directly with `os.ReadFile`, formats temperature from millidegrees Celsius, formats runtime-PM active time, and records read/parse errors per device.
- `main()` refreshes device discovery and readings every second, rebuilds the terminal output, and updates the table in place. A missing device or failed attribute read is displayed as `n/a` with an error rather than as a valid zero value.
- A second goroutine reads from stdin and exits the app when a newline arrives, which is how the CLI is intentionally stopped interactively.
- The repository is Linux-specific and depends on kernel-visible TPU information under `/sys/class/apex`; there is no abstraction layer or config file for device discovery.

## Key codebase conventions

- Keep the app as a small package-main Go CLI unless the scope clearly demands a larger structure.
- Preserve the current Linux/sysfs assumptions: the default path is `/sys/class/apex` and device values are read directly from each device directory, not through an abstraction or API client.
- Keep output in a terminal-friendly table format (`pterm.TableData`), not JSON or other structured output. Preserve visible read errors so the watcher remains useful for diagnosing driver, PCIe, and thermal failures.
- When sorting or enumerating TPU entries, index order is significant; sorting by the device index keeps output stable.
- When adding functionality, avoid introducing broad frameworks or external dependencies unless the project clearly outgrows the current CLI shape.
- The current README indicates TODO items and a small UX scope, so feature work should remain lightweight and inline with the existing command-line monitoring tool.

## Repository context

- Module: `edge-watcher`
- Main package: `package main`
- Dependency: `github.com/pterm/pterm` for terminal rendering
- Runtime assumption: Linux with Edge TPU devices present in `/sys/class/apex`
