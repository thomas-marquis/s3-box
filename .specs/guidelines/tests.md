# Testing Guidelines

This document provides guidelines for writing tests in the `s3-box` project. Following these conventions ensures consistency, readability, and reliability across the codebase.

## 1. General Principles

- **Modern Go Idioms (Go 1.24+)**:
  - Use `t.Context()` when a test needs a context.
  - Use `reflect.TypeFor[T]()` instead of older `reflect.TypeOf` workarounds.
  - Use `b.Loop()` for benchmarks (if any).
- **Package Naming**: Use the `*_test` package for regular test files to ensure they only use the public API (e.g., `package connection_deck_test`).
- **Internal Tests**: To test unexported functions or methods, use `*_internal_test.go` and the same package name as the file under test (e.g., `package connection_deck`).
- **Naming Conventions**: Test functions must follow the `Test<Type>_<Method>` pattern (e.g., `TestDeck_New`).
- **Structure**: Use `t.Run("should ...", func(t *testing.T) { ... })` for subtests to describe expected behavior.
- **Test Workflow**: Structure test bodies with `// Given`, `// When`, `// Then` comments to separate setup, execution, and verification.

## 2. Assertions

- **testify/assert**: Use for all expectations and validation of results.
- **testify/require**: Use ONLY for setup preconditions (e.g., `require.NoError(t, err)`) to stop test execution immediately if the test cannot proceed.
- **Explicit Checks**: Prefer `assert.ErrorIs` for sentinel errors and `assert.Contains` for partial error message matches. Use `assert.JSONEq` for JSON comparisons.

## 3. Types of Tests

### Domain Tests
- Focus on state transitions and business logic.
- Use constructors from domain packages (e.g., `connection_deck.New`).
- Verify state via public methods rather than inspecting internal fields.
- For event-driven logic, call `Notify` with relevant events and assert the resulting state (e.g., `directory.NewLoadSuccessEvent`).

### Infrastructure Tests
- **Integration with S3**: Use `testcontainers-go/modules/localstack` for S3-related tests.
- **Setup Helpers**: Use common setup functions (e.g., `setupS3testContainer`, `setupS3Client`, `setupS3Bucket`) provided in the package's `testutils_test.go`.
- **Cleanup**: Always ensure the container is terminated using `defer terminate()`.
- **Context**: Use `t.Context()` to ensure proper cleanup if the test times out.

### UI Tests (Fyne)
- **Fyne App**: Initialize with `fyne_test.NewApp()` or `fyne_test.NewTempApp(t)`.
- **Rendering**: Use `fyne_test.AssertRendersToMarkup(t, name, canvas)` for visual verification.
- **Interactions**: Use `fyne_test.Tap` or other simulation tools provided by `fyne.io/fyne/v2/test`.
- **Viewmodels**: Mock app context and viewmodels (via `mocks/context` and `mocks/viewmodel`) to isolate the UI layer.

### DTO Tests
- Serialize/deserialize using `encoding/json` and validate with `assert.JSONEq`.

## 4. Mocks

- **go.uber.org/mock (gomock)**: Use for dependencies. Mocks are located in the `mocks/` directory.
- **Generation**: Do NOT edit mocks manually. Mocks are declared in `gen.go`. Run `go generate ./...` to update them.
- **Usage**: Create a controller via `gomock.NewController(t)` and mock expectations using `EXPECT()`. Use `AnyTimes()` for repeated calls where the exact number doesn't matter.

## 5. Test Utilities

- **testutil package**: Leverage the `internal/testutil` package for common fakes (e.g., `FakeAwsConnection`, `FakeS3LikeConnection`, `FakeRandomBucketName`).
- **t.Helper()**: All custom test helpers MUST call `t.Helper()` to keep stack traces accurate and focused on the test call site.

## 6. Code Style in Tests

- **No Comments**: Avoid comments in the code except for `// Given`, `// When`, `// Then` or function documentation. Use readable names instead.
- **Conciseness**: Keep test functions focused. Avoid complex logic or branching in tests.
- **Minimal Mocking**: Prefer real domain objects over mocks when testing domain logic. Mock external infrastructure, UI context, and complex dependencies.

## 7. Run the tests

- **Important** surround the test name with double quotes ("") when it contains simple quotes. For instance:
`go test -v ./internal/infrastructure/s3/ -run "TestNewRepositoryImpl_renameDirectory/should_rename_with_default_grants_when_user_doesn't_have_GetObjectACL_permission"`