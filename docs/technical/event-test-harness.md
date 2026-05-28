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
- **Lazy Publishing**: `Publish(evt)` calls are recorded and only executed when `PrayAndWait` is called.
- **Strict Validation**: The harness ensures that events happen at the correct ticks and that no unexpected events occur during "nothing" ticks.

## Usage

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
th.Expect("(qr)", evtMap).
   Given("'r<-q'", evtMap) 

// When event 'q' is received, the harness will automatically publish 'r' as a followup.
// Since 'q' and 'r' happen at the same tick (T0), we group them in Expect.
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
