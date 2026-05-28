# Event Bus

The Event Bus is the central component for communication between different layers of the application in an event-driven way. It follows the Publish/Subscribe pattern.

## Interface

The `Bus` interface is defined in `internal/domain/shared/event/bus.go`:

```go
type Bus interface {
    Publish(evt Event)
    Subscribe() *Subscriber
}
```

## Implementation

The project provides an in-memory implementation of the `Bus`: `inMemoryBus`.

### InMemoryBus

The `inMemoryBus` manages a set of subscribers and uses a pool of workers to dispatch published events to them.

- **Publishing**: When an event is published, if the payload is a `Carrier`, it is dispatched using the carrier's `Dispatch` method (see [Event Carrier](event-carrier.md)). Otherwise, the event is sent to all subscribers that "Accept" the event based on their registered matchers.
- **Subscribing**: Creating a subscription returns a `Subscriber` object which can be used to register callbacks for specific events.

## Usage

### Publishing an Event

```go
bus.Publish(event.New(MyPayload{...}))
```

### Subscribing to Events

```go
sub := bus.Subscribe().
    On(matcher, func(evt event.Event) {
        // Handle event
    })
sub.ListenWithWorkers(1)
defer sub.Detach()
```
