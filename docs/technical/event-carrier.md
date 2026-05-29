# Event Carrier

An `Event Carrier` is a special type of payload that carries multiple events and defines how they should be dispatched to the `Bus`.

## Interface

```go
type Carrier interface {
    Payload
    Dispatch(bus Bus)
}
```

When a `Bus` receives an event with a `Carrier` payload, it calls `Dispatch` on it instead of broadcasting it to subscribers directly. The `Carrier` implementation then decides how to publish the carried events.

## Type Identification

All carrier types have a special prefix `CarrierTypePrefix = "__carrier__"` that can be used to identify them:

```go
func (t Type) IsCarrier() bool {
    return strings.HasPrefix(t.String(), CarrierTypePrefix)
}
```

## Implementations

### CarriesAll

`CarriesAll` dispatches all carried events to the bus concurrently (up to a configurable maximum concurrency). It waits for a followup event for each dispatched event before being considered "done". Once all followups are received (or it times out), it publishes a "done" event.

**Key Characteristics:**
- Concurrent dispatch with configurable max concurrency (default: 10)
- Waits for followup events for each carried event
- Configurable timeout (default: 60 seconds)
- Configurable done condition (default: accepts any followup)
- Handles duplicate event refs by logging and skipping duplicates

**Configuration Options:**
- `WithMaxConcurrency(n int)`: Sets the maximum number of concurrent workers
- `WithTimeout(d time.Duration)`: Sets the timeout duration
- `WithDoneCondition(cond DoneCondition)`: Sets a custom condition for accepting followups

**Example:**
```go
carrier := event.NewCarriesAll(
    []event.Event{evt1, evt2, evt3},
    func(received []event.Event) event.Event { return doneEvt },
    timeoutEvt,
    event.WithMaxConcurrency(5),
    event.WithTimeout(30 * time.Second),
)
bus.Publish(carrier)
```

### CarriesSequence

`CarriesSequence` dispatches carried events one by one, in sequence. It waits for a followup event for the current event before dispatching the next one.

**Key Characteristics:**
- Sequential dispatch (one at a time)
- Waits for followup before proceeding to next event
- Configurable timeout (default: 60 seconds)
- Configurable done condition (default: accepts any followup)
- Non-blocking Dispatch (spawns a goroutine)
- Empty sequence: Dispatch returns early without publishing

**Configuration Options:**
- `WithTimeout(d time.Duration)`: Sets the timeout duration
- `WithDoneCondition(cond DoneCondition)`: Sets a custom condition for accepting followups

**Example:**
```go
carrier := event.NewCarriesSequence(
    []event.Event{evt1, evt2, evt3},
    func(received []event.Event) event.Event { return doneEvt },
    timeoutEvt,
    event.WithTimeout(10 * time.Second),
)
bus.Publish(carrier)
```

## Done Condition

Both carriers use a `DoneCondition` function to determine if a received event should be considered a valid followup:

```go
type DoneCondition func(sent, received Event) bool
```

The default implementation `DoneWhenFollowupReceived` always returns `true`, meaning any followup event will be accepted. You can provide a custom condition to filter which followups should trigger completion.

**Example of custom done condition:**
```go
carrier := event.NewCarriesAll(
    []event.Event{query},
    func(received []event.Event) event.Event { return doneEvt },
    timeoutEvt,
    event.WithDoneCondition(func(sent, received event.Event) bool {
        // Only accept followups with specific payload
        return received.Payload.(MyPayload).Status == "success"
    }),
)
```

## Use Cases

Carriers are useful for orchestrating complex event flows:

1. **Batch Operations**: Use `CarriesAll` to dispatch multiple independent operations concurrently
2. **Sequential Workflows**: Use `CarriesSequence` when events must be processed in a specific order
3. **Directory Loading**: Load multiple files/directories and wait for all to complete
4. **Error Handling**: Use timeout events to handle cases where operations don't complete

## Testing with TestHarness

When testing carriers, use the `TestHarness` with appropriate marble diagrams:

```go
// Test CarriesAll with followups
h := event.NewTestHarness(bus)
payloads := map[string]event.Payload{
    "e1":  myPayload{"data1"},
    "e2":  myPayload{"data2"},
    "r1":  myPayload{"result1"},
    "r2":  myPayload{"result2"},
    "done": myPayload{"done"},
}

carrier := event.NewCarriesAll(
    []event.Event{e1, e2},
    func(received []event.Event) event.Event { return done },
    timeout,
)

h.Given("('r1<-e1''r2<-e2')", payloads)
h.Expect("('e1''e2''r1''r2')'done'", payloads)
h.Publish(carrier)

assert.True(t, h.PrayAndWait())
```

## Notes

- **Duplicate Refs**: `CarriesAll` checks for duplicate event refs and logs a warning, skipping duplicates. This prevents undefined behavior.
- **Async Dispatch**: The bus calls `Dispatch` asynchronously via `go c.Dispatch(b)`, so carriers are non-blocking from the publisher's perspective.
- **Timeout Behavior**: On timeout, the carrier's `OnTimeout` event is published instead of the done event.
- **Carrier Types**: Both implementations have distinct types: `CarriesAll` returns type `__carrier__.all`, and `CarriesSequence` returns type `__carrier__.sequence`.
