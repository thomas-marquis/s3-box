# Contributing to S3-Box

Thank you for your interest in contributing to S3-Box! This guide will help you get started with setting up your development environment, 
understanding the project structure, and submitting your contributions.

## Setup

### Prerequisites (Linux)

- Go 1.24.6 or later
- `make`
- Git
- Development libraries for Fyne (GUI framework):
  - Ubuntu/Debian: `sudo apt-get install golang gcc libgl1-mesa-dev xorg-dev`
  - Fedora: `sudo dnf install golang gcc libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel libXxf86vm-devel`
  - Arch: `sudo pacman -S go gcc libx11 libxcursor libxrandr libxinerama mesa libxi libxxf86vm`

### Installing a Development Environment

1. **Clone the repository:**
   ```bash
   git clone https://github.com/thomas-marquis/s3-box.git
   cd s3-box
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Build the application:**
   ```bash
   make build
   ```

4. **Run the application:**
   ```bash
   make run
   ```

5. **Run tests:**
   ```bash
   make test
   ```
   or, for a full check (tests + linter):
   ```bash
   make check
   ```

6. **Generate mocks (if needed):**
   ```bash
   go generate ./...
   ```

## Process

### Reporting Issues

If you find a bug or want to request a feature:

1. Check if the issue already exists in the [Issues](https://github.com/thomas-marquis/s3-box/issues) section
You can also check the GitHub Project board [here](https://github.com/users/thomas-marquis/projects/8)
2. If not, create a new issue with:
   - A clear, descriptive title
   - A detailed description of the problem or feature
   - Steps to reproduce (for bugs)
   - Expected vs. actual behavior (for bugs)
   - Your environment details (OS, Go version, etc.)

### Creating a Pull Request

1. **Fork the repository** and create a new branch from `main`:
   ```bash
   git checkout -b feat/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```

2. **Make your changes** following the coding conventions (see below)

3. **Write or update tests** for your changes

4. **Ensure all tests pass:**
   ```bash
   make check
   ```

5. **Commit your changes** with clear, descriptive commit messages

6. **Push to your fork:**
   ```bash
   git push origin feat/your-feature-name
   ```

7. **Open a Pull Request** on GitHub:
   - Provide a clear title and description
   - Reference any related issues using `#issue-number` in the PR description (e.g., "Fixes #42" or "Closes #123")
   - Describe what changes you made and why
   - Include screenshots/GIFs if the changes affect the UI

8. **Address review feedback** if requested

## Project Architecture

S3-Box follows clean architecture principles with a layered design approach:

### Layers

The project is organized into the following layers, from innermost to outermost:

```
infrastructure → domain ← viewmodel ← view
```

#### Domain Layer (`internal/domain`)

The core business logic, independent of UI and infrastructure concerns.

- **Location:** `internal/domain/`
- **Responsibility:** Core models, business rules, domain events, and interfaces
- **Key concepts:**
  - Root entities: `connection_deck.ConnectionDeck` and `directory.Directory`
  - Domain events flow through a shared event bus interface
  - Follows Domain Driven Design (DDD) principles: entities, aggregates, value objects, domain services

#### ViewModel Layer (`internal/ui/viewmodel`)

Acts as a use case layer and bridges between the domain and the view.

- **Location:** `internal/ui/viewmodel/`
- **Responsibility:** Coordinates domain logic with infrastructure services
- **Key concepts:**
  - Wraps domain entities
  - Orchestrates use cases
  - Follows MVVM (Model-View-ViewModel) pattern

#### View Layer (`internal/ui/views`)

The presentation layer using the Fyne GUI framework.

- **Location:** `internal/ui/views/`
- **Responsibility:** Renders widgets and handles user interactions
- **Key concepts:**
  - Fyne widgets and UI components
  - Theme and resource assets

#### Infrastructure Layer (`internal/infrastructure`)

External concerns like storage, APIs, and third-party integrations.

- **Location:** `internal/infrastructure/`
- **Responsibility:** Implements repositories and adapters
- **Key concepts:**
  - S3 access via AWS SDK v2
  - Fyne preferences storage
  - DTO mappings
  - Notification publishing

### Entry Point

- **Location:** `main.go`
- **Responsibility:** Builds the Fyne app and wires dependencies via `internal/ui/app`

### Event-Driven Architecture

- Events flow through the shared `event.Bus` interface
- Concrete implementation: `internal/ui/app/event_bus.go`

### Clean Architecture Principles

- The domain layer is independent of UI and infrastructure layers
- Dependencies point inward: **infrastructure → domain ← viewmodel ← view**
- Business logic remains testable and isolated from external frameworks

## Coding Conventions

For detailed coding conventions, testing guidelines, and best practices, please refer to the [Guidelines](.junie/guidelines.md) document.

### Key Points

- Use Go idioms and follow standard Go formatting (`gofmt`, `goimports`)
- Write tests using `testing` with `testify/assert` and `testify/require`
- Structure tests with `t.Run("should ...")` subtests
- Use `// Given`, `// When`, `// Then` comments for test readability
- Generate mocks using `gomock` and `go generate` (never edit mocks manually)
- Follow the layered architecture strictly—avoid circular dependencies
- Prefer explicit error handling

## Key links and documentation

* [Fyne documentation](https://docs.fyne.io/)
* [Project's board](https://github.com/users/thomas-marquis/projects/8/views/1)

---

Thank you for contributing to S3-Box! Your efforts help make this project better for everyone.
