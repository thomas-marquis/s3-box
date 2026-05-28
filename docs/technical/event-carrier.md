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

## Implementations

### CarriesAll

Dispatches all carried events to the bus. By default, it dispatches them concurrently (up to a maximum concurrency).
It waits for a followup event for each dispatched event before being considered "done". Once all followups are received (or it times out), it publishes a "done" event.

### CarriesSequence

Dispatches carried events one by one. It waits for a followup event for the current event before dispatching the next one in the sequence.

## Usage

Carriers are useful for orchestrating complex event flows, such as loading a directory which might involve multiple S3 requests.

```go
carrier := event.NewCarriesSequence(
    []event.Event{evt1, evt2},
    func(received []event.Event) event.Event { return doneEvt },
    timeoutEvt,
)
bus.Publish(carrier)
```
