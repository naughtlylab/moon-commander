# Project Governance & Engineering Rules

## 1. Core Philosophy
This is a strictly spec-driven Go project using Bubble Tea (`charmbracelet/bubbletea`) and Lip Gloss (`charmbracelet/lipgloss`). You must NEVER "vibe code," guess requirements, or write untested UI/filesystem coupling.

## 2. Mandatory Development Workflow (Vertical Slices)
For every feature or milestone, you must execute these steps in strict order:
1. **Types & Spec Check:** Verify the required structs, interfaces, and `tea.Msg` types exist in `SPEC.md`. If they do not, propose an update to `SPEC.md` and STOP for human approval.
2. **Red (Tests First):** Write table-driven unit tests in `*_test.go` testing the `Update` function and state transitions using mock `FileSystem` interfaces. Confirm the tests fail (`go test ./...`).
3. **Green (Implementation):** Write the minimal implementation code required to pass the tests.
4. **Verify:** Run `go test -v -race ./...` and `go vet ./...`. Fix all warnings before reporting completion.

## 3. Architectural Guardrails
* **Zero I/O in `Update` or `View`:** Never call `os.*`, `io.*`, or `filepath.*` disk operations directly inside `Update()` or `View()`. All disk operations must be executed asynchronously inside `tea.Cmd` functions via the injected `FileSystem` interface.
* **Pure State Transitions:** Given a `Model` and a `tea.Msg`, `Update()` must be 100% deterministic.
* **No Global State:** All state lives inside the root `Model` struct.
* **File Size Limit:** No single `.go` file may exceed 300 lines. Split panes, commands, styles, and messages into dedicated files.

## 4. Commands
* Run tests: `go test -v -race ./...`
* Vet code: `go vet ./...`
* Run app: `go run ./cmd/fm`