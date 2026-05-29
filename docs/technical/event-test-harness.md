# Event Test Harness

The `TestHarness` is a utility to simplify testing of event-driven flows. It allows defining expected sequences of events and providing automatic responses to them using a marble diagram syntax that accounts for relative timing.

## Features

- **Expect**: Define a sequence of events and the ticks at which they are expected. Must be called exactly once.
- **Given**: Define timed event publications or explicit followup rules. Can be called at most once.
- **Marble Syntax**: 
    - `-`: A single time tick where nothing should happen.
    - `___`: A contiguous sequence of underscores counts as exactly one time tick (useful for visual alignment).
    - `a`, `b`, `'label'`: Events. Each event also occupies one time tick in the diagram.
    - `(ab)`: Grouping. Multiple events or followup rules can happen at the same tick.
    - `'r<-e'`: Explicit followup rule (only in `Given`). Specifies that `r` should be published when `e` is received.
- **Delayed Followup Activation**: Followup rules specified in `Given` can have delayed activation times based on their position in the marble string. The harness will retroactively check if triggers were already received before the rule became active.
- **Lazy Publishing**: `Publish(evt)` calls are recorded and only executed when `PrayAndWait` is called.
- **Strict Validation**: The harness ensures that events happen at the correct ticks and that no unexpected events occur during "nothing" ticks.
- **Detailed Error Reporting**: When `PrayAndWait()` returns `false`, use `GetFailureReason()` to get a detailed explanation of what went wrong.

## Usage

### Guidelines

- declare the marble templates in dedicated variable and ensure everything is correctly visually aligned for readability

### Basic Example

```go
th := event.NewTestHarness(bus, 
    event.WithTickDuration(10 * time.Millisecond),
).Expect("a-b", map[string]event.Payload{
    "a": payloadA{},
    "b": payloadB{},
})

// 'a' will be published at the start of PrayAndWait (Tick 0)
th.Publish(event.New(payloadA{}))

// 'b' will be published by the harness at Tick 2
th.Given("--b", map[string]event.Payload{"b": payloadB{}})

if !th.PrayAndWait() {
    t.Fatal("Harness failed: timeline mismatch")
}
```

### Simultaneous Events and Groups

Use the `(...)` syntax to define events happening at the same tick.

```go
exp := "  (qr)"
given := "'r<-q'"
th.Expect(exp, evtMap).
   Given(given, evtMap)

// When event 'q' is received, the harness will automatically publish 'r' as a followup.
// Since 'q' and 'r' happen at the same tick (T0), we group them in Expect.
```

### Delayed Followup Activation

Followup rules can be activated at specific ticks, allowing modeling of real-world scenarios where responses don't arrive immediately:

```go
// Followup rules become active at different times
given := "--('r1<-e1''r2<-e2')--'r3<-e3'"
// e1, e2, e3 are published at T0, but:
// - r1 and r2 followup rules become active at T2
// - r3 followup rule becomes active at T4
// The harness will retroactively trigger r1/r2 if e1/e2 were already received

exp := "('e1''e2''e3')--('r1''r2')--'r3'"
// Events are expected at: T0 (e1,e2,e3), T2 (r1,r2), T4 (r3)

th.Given(given, payloads).
   Expect(exp, payloads).
   Publish(carrier)
```

## Internal Mechanism

- **Parser**: Uses an "op code" approach to convert marble strings into a sequence of Ticks, Events, Followups, and Groups.
- **Timeline**: `Expect` and `Given` build a timeline of events and actions indexed by tick number.
- **Simulation**: `PrayAndWait` starts a timer. At each tick, it publishes scheduled events.
- **Validation**:
    - Every received event is recorded with its arrival tick.
    - `PrayAndWait()` returns `true` only if all expected events arrived at the correct ticks AND no unexpected events occurred.
    - Any received event not explicitly expected at its arrival tick causes a failure.

## Configuration

The harness is configured using the functional option pattern in `NewTestHarness`:

- `event.WithTickDuration(d)`: Sets the duration of a single tick (default 10ms).
- `event.WithHarnessTimeout(d)`: Sets the maximum wait time for expectations (default 10s).
