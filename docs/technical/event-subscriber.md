# Event Subscriber

A `Subscriber` is used to listen for events published on the `Bus`.

## Key Features

- **Matchers**: Subscribers use `Matcher` objects to filter which events they are interested in.
- **Callbacks**: Functions can be registered to be executed when a matching event is received.
- **Concurrency**: Supports listening with multiple workers or in a non-blocking way.

## Usage

### Registering Callbacks

Use the `On` method to register a callback for a specific matcher:

```go
sub := bus.Subscribe().
    On(event.TypeMatcher("my.event.type"), func(evt event.Event) {
        // Handle event
    })
```

### Starting to Listen

After registering callbacks, you must start listening:

- `ListenWithWorkers(n)`: Starts `n` goroutines to process events sequentially from the internal channel.
- `ListenNonBlocking()`: Starts a goroutine that spawns a new goroutine for each matching event.

### Detaching

Always call `Detach()` to stop the subscriber and clean up resources:

```go
defer sub.Detach()
```

## Internal Mechanism

The `Subscriber` maintains a map of `Matcher` to callback functions. When an event is received on its internal channel, it iterates through the matchers and executes the corresponding callbacks for those that match.
