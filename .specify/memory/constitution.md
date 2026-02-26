# S3-Box Constitution

## Core Principles

### I. Architecture & Dependency Direction
S3-Box follows clean, layered architecture with strict dependency direction: `infrastructure → domain ← viewmodel ← view`. The domain is independent of UI and infrastructure. Root entities live in `internal/domain` (notably `connection_deck.ConnectionDeck` and `directory.Directory`). Viewmodels orchestrate use cases and expose data to views; views render Fyne widgets only. No layer may bypass or invert these boundaries.

### II. Event-Driven Domain Flow
State changes in the domain are expressed as domain events and flow through the shared `event.Bus` interface. Repositories and adapters subscribe and publish through the bus; the concrete bus lives in `internal/ui/app/event_bus.go`. Domain state transitions occur via `Notify(...)` on domain entities with success/failure events.

### III. Test Discipline (Non-Negotiable)
All behavioral changes require tests. Use Go's `testing` with `testify/assert` and `testify/require`. Structure tests with `t.Run("should ...")` and `// Given`, `// When`, `// Then` comments. Exported behavior is tested from `*_test` packages; internal-only tests use `*_internal_test.go` with the same package name. Mocks are generated with `gomock` and `go generate`; generated files under `mocks/` are never edited by hand.

### IV. Infrastructure Boundaries
Infrastructure code lives in `internal/infrastructure` and implements repositories/adapters or event listeners. S3 access uses AWS SDK v2 and DTO mappings live in the infrastructure layer. Integration tests for S3 rely on `testcontainers-go/modules/localstack` and belong to infrastructure tests. Domain code must not depend on infrastructure or UI concerns.

### V. UI & UX Consistency
The UI is built with Fyne. Views render widgets and delegate behavior to viewmodels; viewmodels expose bindings and orchestrate domain + infrastructure. UI tests use `fyne_test.NewApp()`/`fyne_test.NewTempApp(t)` and rendering assertions via `fyne_test.AssertRendersToMarkup`.

## Technology & Constraints

- Language and runtime: Go (project targets Go 1.24.6+).
- UI framework: Fyne; use Fyne preferences for settings storage.
- Cloud access: AWS SDK v2 for S3 and S3-compatible providers.
- Formatting: `gofmt`/`goimports` standards are mandatory.
- Generated code: mocks in `mocks/` are generated via `go generate`; never edit manually.
- Eventing: all cross-layer coordination uses the shared `event.Bus` interface.

## Workflow & Quality Gates

- Run `make test` for local validation; `make check` (tests + lint) is the default quality gate for PRs.
- New or changed behavior must include tests that follow the project testing guidelines.
- UI changes require updated screenshots or GIFs in PRs when behavior or layout changes are visible.
- Architecture changes must update documentation and remain consistent with `internal/domain`, `internal/ui`, and `internal/infrastructure` boundaries.
- Follow `.junie/guidelines.md` for detailed conventions and testing rules.

## Governance
This constitution supersedes other practices. Any amendment must update this document, include a rationale, and keep it consistent with the architecture and testing guidelines.

**Version**: 1.0.0 | **Ratified**: February 26, 2026 | **Last Amended**: February 26, 2026
