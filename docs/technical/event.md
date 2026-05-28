# Event

An `Event` is a data structure representing something that happened in the system.

## Structure

```go
type Event struct {
    ID        string          // Unique identifier for the event
    Payload   Payload         // The data carried by the event
    Context   context.Context // Execution context
    Ref       string          // Reference ID to link events (e.g., for followups)
    eventType Type            // The type of the event
}
```

## Payload

Every event must have a `Payload`. A `Payload` is an interface that returns the event type:

```go
type Payload interface {
    Type() Type
}
```

## Creating Events

### New Event

Use `event.New` to create a new event with a payload:

```go
evt := event.New(MyPayload{})
```

### Followup Event

A followup event is an event that is linked to a previous event via the `Ref` field. Use `event.NewFollowup`:

```go
followup := event.NewFollowup(originalEvent, ResponsePayload{})
```

## Options

Event creation can be customized using `Option` functions, such as `WithContext`.
