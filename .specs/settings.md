# Settings Entity Specification

## Overview

The `Settings` entity manages application settings with type safety and event-driven persistence. It follows a clear lifecycle that separates registration, loading, reading, and saving operations.

**Thread Safety:** All public methods are thread-safe. The entity uses internal mutex locks and a state pattern to protect its state from concurrent access.

## Entity States

The entity uses a state pattern to manage its lifecycle:

- **IdleState**: Initial state. All operations are permitted.
- **LoadingState**: Load operation in progress. Only Read operations are permitted. Register, Write, Save, and Load operations return `ErrNotReady`.
- **SavingState**: Save operation in progress. Register, Read, and Write operations are permitted. Save and Load operations return `ErrNotReady`.

The entity transitions between states as follows:
- `IdleState` → `LoadingState`: When `Load()` is called
- `LoadingState` → `IdleState`: When `Notify(LoadSucceeded)` or `Notify(LoadFailed)` is called
- `IdleState` → `SavingState`: When `Save()` is called (only if there are pending events)
- `SavingState` → `IdleState`: When `Notify(SaveSucceeded)` or `Notify(SaveFailed)` is called

## Lifecycle

### 1. Instantiation

```go
s := settings.NewSettings()
```

### 2. Registration

```go
s.Register(
    settings.AString("app.theme", "dark"),
    settings.AUint64("app.maxRetries", 5),
    settings.ADuration("app.timeout", 30*time.Second),
)
```

**Constraints:**
- Duplicate registration returns `ErrAlreadyExists`
- Empty/whitespace name returns `ErrInvalidType`
- Permitted in `IdleState` and `SavingState`
- Returns `ErrNotReady` in `LoadingState`

### 3. Loading

```go
evt, err := s.Load()
if err != nil {
    // Handle error (ErrNotReady if not in IdleState)
}
bus.Publish(evt)
```

The `LoadTriggered` event signals to infrastructure to load registered settings.

### 4. Notification

```go
s.Notify(responseEvent)
```

- `LoadSucceeded`: Transitions to `IdleState`, merges values
- `LoadFailed`: Transitions to `IdleState`
- `SaveSucceeded`: Transitions to `IdleState`
- `SaveFailed`: Transitions to `IdleState`, restores pending events
- `WriteSucceeded`: Updates state, triggers observers

### 5. Reading Values

```go
val := s.ReadString("app.theme")
```

- Returns stored value or default
- Panics with `ErrUnregistered` if not registered
- Permitted in all states

### 6. Writing Values

```go
err := s.Write("app.theme", "light")
```

- Returns `ErrUnregistered`, `ErrInvalidType`, or `ErrNotReady` (LoadingState)
- Permitted in `IdleState` and `SavingState`
- **Duplicate writes**: When `Write()` is called multiple times for the same setting name, only the latest value is kept in pending events. Previous pending events for the same setting are replaced.

### 7. Saving

```go
evt, err := s.Save()
if err != nil {
    // Handle error (ErrNotReady if LoadingState or SavingState)
}
bus.Publish(evt)
```

- Returns `(event.Event, error)`
- Returns `ErrNotReady` if in `LoadingState` or `SavingState`
- Returns `SaveSucceeded` if no pending events
- Returns `carrier.All` event and transitions to `SavingState` if pending events

### 8. Canceling

```go
s.Cancel()
```

- Clears all pending write events
- Transitions to `IdleState`
- Permitted in all states

### 8. Observer Pattern

```go
unobserve := s.Observe("app.theme", func(value any) {
    // WARNING: Callback MUST be non-blocking and short.
    // Panics will NOT be recovered.
})
defer unobserve()
```

- Callbacks are synchronous
- No panic recovery

## State Machine

```
IdleState:
    Register: ✓
    Read:    ✓
    Write:   ✓
    Load:    → LoadingState
    Save:    → SavingState (if pending)

LoadingState:
    Register: ✗ (ErrNotReady)
    Read:    ✓
    Write:   ✗ (ErrNotReady)
    Load:    ✗ (ErrNotReady)
    Save:    ✗ (ErrNotReady)
    
SavingState:
    Register: ✓
    Read:    ✓
    Write:   ✓
    Load:    ✗ (ErrNotReady)
    Save:    ✗ (ErrNotReady)
```

## Errors

- `ErrAlreadyExists` - duplicate registration
- `ErrUnregistered` - read/write unregistered setting
- `ErrInvalidType` - type mismatch or empty name
- `ErrNotReady` - operation not permitted in current state
- `ErrTimeout` - timeout (in infrastructure events)

## Storage and Migration

The settings are stored in Fyne preferences under the key `settingsV2`. 

**V1 Format**: The legacy format stored settings as a simple struct with fields like `TimeoutInSeconds`, `MaxFilePreviewSizeBytes`, and `ColorTheme`.

**V2 Format**: The current format stores settings as a map of setting names to `settingDTO` objects, which include both the value and type information.

**Migration Strategy**: When a user upgrades from a V1 version:
- If the `settingsV2` key doesn't exist or is empty, the handler should initialize an empty map
- No automatic migration from V1 to V2 is performed
- The user will experience a "reset" of their settings to defaults
- This is acceptable as a trade-off for simplicity

## Implementation Notes

Use a state pattern with state types implementing operation permission checks.
