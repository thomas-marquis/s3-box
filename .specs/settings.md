# SettingsV3 Entity Specification

## Overview

The `SettingsV3` entity manages application settings with type safety and event-driven persistence. It follows a clear lifecycle that separates registration, loading, reading, and saving operations.

**Thread Safety:** All public methods are thread-safe. The entity uses internal mutex locks to protect its state from concurrent access.

## Lifecycle

### 1. Instantiation

The entity is created using the `NewSettingsV3()` constructor. This initializes an empty settings container with:
- Empty registration map
- Empty values map
- `isLoading` flag set to `false`
- Empty pending events list
- Empty observers map

```go
s := settings.NewSettingsV3()
```

### 2. Registration

Settings are registered with a name, a type, and a default value. This is a local operation that does NOT emit any events. Registration must occur before loading.

The available types are:
- `StringType` - for string values
- `Uint64Type` - for uint64 values
- `DurationType` - for time.Duration values

Registration uses the `Register()` method with `Registration` functions:
- `AString(name, defaultValue string)`
- `AUint64(name string, defaultValue uint64)`
- `ADuration(name string, defaultValue time.Duration)`

```go
s.Register(
    settings.AString("app.theme", "dark"),
    settings.AUint64("app.maxRetries", 5),
    settings.ADuration("app.timeout", 30*time.Second),
)
```

**Constraints:**
- Duplicate registration (same name) returns `ErrAlreadyExists` error
- Registration is possible when `isLoading` is `false` (allowed before first load and after any load completes)
- Registration stores both the default value and the type information

### 3. Loading

The first load operation triggers the `LoadTriggered` event. The entity creates this event but does NOT send it to the bus - the client code is responsible for publishing it.

```go
evt, err := s.Load()
// Client must publish the event
bus.Publish(evt)
```

The `LoadTriggered` event contains no payload - it signals to the infrastructure that registered settings should be loaded.

**Note:** The client code is responsible for publishing all events. The entity assumes events will be published and does not implement timeout mechanisms.

#### Infrastructure Responsibilities

The infrastructure layer receives the `LoadTriggered` event and must:

1. **Retrieve all registered settings** - The event implies which settings are registered based on the entity's state
2. **Handle various scenarios:**

   **3.1 All registered values exist on storage**
   - All requested settings are present with compatible types
   - Return all values in `LoadSucceeded` event

   **3.2 Partial match**
   - Some registered values exist, some don't
   - For missing values: create them in storage with their default values (provided during registration)
   - Return ALL registered values (both existing and newly created) in `LoadSucceeded` event

   **3.3 Type incompatibility**
   - A stored value has a type that doesn't match the registered type
   - Emit `LoadFailed` event with an explicit error
   - If multiple values have type errors, the error message must list ALL of them

   **3.4 Extra values on storage**
   - Storage contains additional settings not registered by this entity
   - These extra values MUST be ignored
   - They belong to other `SettingsV3` entities and must not be affected
   - Only return the registered values in the success event

The `LoadSucceeded` event payload contains:
- `Values`: map[string]any - the actual values for all registered settings
- `Registered`: map[string]SType - the types for all registered settings

The `LoadFailed` event payload contains:
- `Err`: error - describes the failure, including all type mismatches if applicable

### 4. Notification

The client must call `s.Notify(evt)` with the response event from the infrastructure.

When `Notify` receives a `LoadSucceeded` event:
- Sets `isLoading` to `false`
- Merges the received values with registered ones
- Only stores values that:
  - Are in the `Registered` map
  - Have matching types
- Converts `int64` nanoseconds to `time.Duration` for duration types
- Ignores any values not registered with this entity

At this stage, the entity contains ONLY the registered values in its memory.

### 5. Reading Values

After successful loading, values can be read using the type-specific methods:
- `ReadString(name string) string`
- `ReadUint64(name string) uint64`
- `ReadDuration(name string) time.Duration`

These methods:
- Return the in-memory stored value
- Panic with `ErrUnregistered` if the setting name is not registered
- Return the default value if the setting was never loaded (value not updated by LoadSucceeded)

Read operations are always permitted, even during load/reload.

### 6. Reloading

The user can call `Load()` at any time to refresh the entity's content. This is useful when settings may have been changed remotely.

**Important:** When loading is in progress (`isLoading == true`):
- Read operations are still permitted
- Write operations MUST return an error
- Registration operations MUST return an error
- Save operations MUST return an error

### 7. Writing Values

To update a setting value, use the `Write()` method:

```go
s.Write("app.theme", "light")
```

The `Write` method:
- Validates the setting is registered
- Validates the value type matches the registered type
- Adds a `WriteTriggered` event to the pending events list
- **Does NOT update the internal state** - state is updated only when `Notify(WriteSucceeded)` is called
- Returns `ErrUnregistered` if the setting is not registered or type doesn't match
- Returns `ErrNotReady` if `isLoading` is `true` (load in progress)

Multiple write calls stack events in the pending list.

**Important:** The internal value for a setting is only updated when the corresponding `WriteSucceeded` event is received via `Notify()`. Until then, read operations return the previous value (or default if never loaded).

### 8. Saving

When the user wants to persist pending changes, call the `Save()` method:

```go
evt := s.Save()
// Client must publish the event
bus.Publish(evt)
```

The `Save` method:
- Returns an event that wraps and orchestrates all pending events
- Clears the pending events list

If there are no pending events, returns `SaveSucceeded` immediately.

If there are pending events:
- Returns a `carrier.All` event (domain-layer construct) that:
  - Executes all pending `WriteTriggered` events
  - On success for each write: infrastructure **MUST** publish `WriteSucceeded` for that setting
  - On success for all writes: infrastructure **MUST** also publish `SaveSucceeded`
  - On failure for any write: infrastructure **MUST** publish `SaveFailed` with the list of failed events
- The pending events list is emptied after `Save()` is called

**Notification Flow:**
- Client must call `s.Notify(WriteSucceeded{Name: "setting", Value: value})` for each successful individual write
  - This updates the entity's internal state for that setting
  - This triggers observer callbacks for that setting
- Client must call `s.Notify(SaveSucceeded{})` after all `WriteSucceeded` notifications

If save fails, the client should call `s.Notify(SaveFailed{...})` which:
- Restores the pending events (so they can be retried)
- Does NOT update any internal state or notify observers

**Note:** The `isLoading` flag is only affected by `Load()` and `Notify()` calls for load events. Save operations do not modify `isLoading`.

## Observer Pattern

The entity supports observation of setting changes:

```go
unobserve := s.Observe("app.theme", func(value any) {
    // Called when app.theme value changes
})
defer unobserve()
```

Observers are notified when:
- A `WriteSucceeded` event is received via `Notify()` for the observed setting name
- The internal state for that setting has been updated

Observers are NOT notified for:
- Registration
- Load operations
- Save operations (only individual `WriteSucceeded` triggers notifications)
- `SaveSucceeded` or `SaveFailed` events

## State Machine

```
Initial (isLoading=false)
    │
    ▼
Register settings (local operation, no events)
    │
    ▼
Load() → LoadTriggered event emitted, isLoading=true
    │
    ▼
Client publishes event → Infrastructure processes
    │
    ├── LoadSucceeded → Notify() → isLoading=false, values stored
    │
    └── LoadFailed → Notify() → isLoading=false, error stored
    │
    ▼
Idle (isLoading=false)
    │
    ├── Read*() → returns values
    │
    ├── Write() → adds WriteTriggered to pending events (state unchanged)
    │
    ├── Register() → allowed (adds new settings)
    │
    ├── Load() → isLoading=true, returns LoadTriggered
    │
    └── Save() → returns carrier.All event
         │
         ▼
         Client publishes → Infrastructure processes
         │
         ├── WriteSucceeded (per setting) → Notify() → state updated, observers called
         │
         ├── SaveSucceeded → Notify() → no state change
         │
         └── SaveFailed → Notify() → pending restored
```

## Error Handling

All errors are **sentinel errors** and can be checked using `errors.Is()`:
- `ErrAlreadyExists` - returned when attempting to register a setting name that already exists
- `ErrUnregistered` - returned when attempting to read or write a setting that was never registered (used in panics for read operations, returned as error for write operations)
- `ErrTimeout` - returned when a save operation fails due to timeout
- `ErrInvalidType` - returned when a value type doesn't match the registered type for a setting
- `ErrNotReady` - returned when attempting to write, register, or save while `isLoading` is `true` (load in progress)

## Edge Cases

### Registration
- **Empty or whitespace-only name**: Registration MUST return an error for invalid names
- **Zero registrations**: Calling `Load()` with no registered settings MUST return a `LoadTriggered` event. Infrastructure MUST return empty `Values` and `Registered` maps in `LoadSucceeded`
- **Concurrent registration**: Thread-safe - concurrent calls to `Register()` are serialized via mutex
- **Register after Load**: MUST be allowed when `isLoading` is `false` (after any successful or failed load)

### Loading
- **Storage returns nil values**: `Notify(LoadSucceeded)` MUST handle `nil` values in the `Values` map by using the registered default value
- **All values missing**: Infrastructure MUST create all missing settings with their registered default values and return them in `LoadSucceeded`
- **Storage unreachable**: Infrastructure MUST return `LoadFailed` with an appropriate error
- **Concurrent Load() calls**: Thread-safe - only one load operation can be in progress; subsequent calls return `ErrNotReady` while `isLoading` is `true`

### Reading
- **Read before Load**: Returns the registered default value
- **Read during Load**: Permitted - returns the last known value (or default if never loaded)
- **Read unregistered setting**: Panics with `ErrUnregistered`
- **Concurrent reads**: Thread-safe - multiple concurrent read operations are permitted

### Writing
- **Write during Load**: MUST return `ErrNotReady`
- **Write unregistered setting**: MUST return `ErrUnregistered`
- **Write with nil value**: MUST return an error for nil values (except for string type where empty string is valid)
- **Write same setting multiple times**: Each write adds a separate `WriteTriggered` event to pending; all are processed in order during Save
- **Write with wrong type**: MUST return `ErrInvalidType` if the value type doesn't match the registered type
- **Concurrent writes**: Thread-safe - write operations are serialized

### Saving
- **Save with no pending writes**: Returns `SaveSucceeded` immediately without changing state
- **Save during Load**: MUST return `ErrNotReady` (cannot save while `isLoading` is `true`)
- **Save while already saving**: MUST be allowed (Save does not set `isLoading`)
- **Concurrent Save() + Write()**: Both MUST be allowed (Save does not set `isLoading`)
- **Save failure recovery**: On `Notify(SaveFailed)`, pending events are restored; client can retry

### Observer Pattern
- **Observe unregistered setting**: Returns an unobserve function but never calls the callback (silent no-op) - alternative: could return error
- **Observe same setting multiple times**: Each observation gets its own callback; all are called when the setting changes
- **Callback panics**: The entity MUST recover from panics in observer callbacks and continue notifying other observers
- **Double unobserve**: Calling the unobserve function multiple times MUST be a no-op (no panic, no error)
- **Unobserve during notification**: Safe - the unobserve function can be called from within the callback

### Type Handling
- **Zero values**: All zero values (empty string, 0, 0 duration) are valid and MUST be stored correctly
- **Negative duration**: Valid Go `time.Duration`; MUST be accepted and stored correctly
- **Maximum uint64**: MUST be accepted and stored correctly
- **Very long strings**: MUST be accepted (no length limit)

### Event Handling
- **Notify with unexpected event type**: MUST be a no-op or return an error (implementation choice, but must be documented)
- **LoadSucceeded with mismatched Registered map**: Values not in Registered map MUST be ignored; values in Registered map but not in Values map MUST use default values
- **WriteSucceeded for unregistered setting**: MUST be ignored (silent no-op)

## Key Principles

1. **Separation of Concerns**: Domain entity manages state, infrastructure handles persistence
2. **Type Safety**: All values have explicit types, mismatches are caught early
3. **Event-Driven**: All persistence operations go through the event bus
4. **Isolation**: Multiple SettingsV3 entities can coexist without interfering with each other's storage
5. **Atomicity**: Save operations handle all pending writes as a unit
6. **Thread Safety**: All public methods are safe for concurrent use via internal mutex protection

## Testing Guidelines

Tests must follow the project's constitution testing guidelines (`testify/assert`, `testify/require`, `t.Run`, Given/When/Then comments).

### Domain Tests
- Use `settings.NewSettingsV3()` to create the entity
- Test registration with `AString`, `AUint64`, `ADuration`
- Test error cases: `ErrAlreadyExists`, `ErrUnregistered`, `ErrNotReady`, `ErrInvalidType`
- Use `assert.ErrorIs` for sentinel error checks
- Verify state transitions via `isLoading` flag (may need internal test access)
- Test observer callbacks are triggered correctly

### Event Flow Tests
- Test `Load()` → `LoadTriggered` event creation and `isLoading` set to true
- Test `Notify(LoadSucceeded)` → state updates and `isLoading` set to false
- Test `Write()` → `WriteTriggered` in pending list
- Test `Save()` → `carrier.All` event creation
- Test `Notify(WriteSucceeded)` → state update and observer notification
- Test `Notify(SaveSucceeded)` → no state change
- Test `Notify(SaveFailed)` → pending events restored

### Thread Safety Tests
- Use `t.Run` with parallel subtests
- Run concurrent `Register()`, `Write()`, `Read*()`, `Load()`, `Save()` operations
- Verify no race conditions (use `-race` flag)
- Verify state consistency after concurrent operations

### Edge Case Tests
- Test all edge cases listed in the Edge Cases section
- Test nil values, empty strings, zero values
- Test concurrent access patterns
- Test error conditions and recovery

---

## Implementation Plan

### Current State
The `SettingsV3` entity exists in `internal/domain/settings/settings_v3.go` with:
- Basic structure, type system (SType), and event types
- Registration, Write, Read, Load, Save, Notify, Observe methods
- Thread safety via `sync.RWMutex` for some operations
- Current flag: `isReady` (needs replacement with `isLoading`)

### Required Changes

#### 1. State Management Refactor
- Replace `isReady` field with `isLoading` in the `SettingsV3` struct
- Initialize `isLoading = false` in constructor
- Remove `IsReady()` method (no longer needed)

#### 2. Error Types
- Add `ErrInvalidType` for type mismatch validation
- Add `ErrNotReady` for operations blocked during load

#### 3. Load/Notify Flow
- `Load()`: Set `isLoading = true` (with mutex), return `LoadTriggered`
- `Notify(LoadSucceeded)`: Set `isLoading = false`, merge values
- `Notify(LoadFailed)`: Set `isLoading = false`, store error

#### 4. Write Method
- Add `isLoading` check: return `ErrNotReady` if true
- Add type validation: return `ErrInvalidType` if value type doesn't match registered type
- Keep existing pending event addition

#### 5. Register Method
- Add `isLoading` check: return `ErrNotReady` if true
- This allows registration **after** load completes (isLoading=false)

#### 6. Save Method
- Remove `isReady = false` setting (Save doesn't affect isLoading)
- Keep pending events clearing logic
- Return `carrier.All` with proper success/failure handlers

#### 7. WriteSucceeded Handling
- Validate setting is registered
- Validate type matches
- Update internal state
- Trigger observer callbacks

#### 8. Thread Safety
- Ensure all `isLoading` reads/writes are mutex-protected
- Review all public methods for proper lock coverage

#### 9. Tests Update
- Update tests to reflect `isLoading` behavior
- Add tests for registration after load
- Add tests for concurrent access
- Update error assertions to include `ErrNotReady` and `ErrInvalidType`
