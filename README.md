# PubSub Package

This repository contains a PubSub package with two implementations that can be used in your Go projects, including applications like GoGin with SSE (Server-Sent Events). The package provides an in-memory PubSub system and a MongoDB Adapter PubSub system for reliable event handling across distributed systems.

## Contents

1. **In-Memory PubSub** - A simple, lightweight in-memory publish-subscribe (PubSub) system suitable for local use cases where persistence is not required.
2. **MongoDB Adapter PubSub** - A MongoDB-backed PubSub system that ensures messages are persisted and can be consumed by multiple subscribers, even across distributed services.

## In-Memory PubSub

The in-memory PubSub implementation allows for event-based communication between parts of your application. It allows you to subscribe to events, publish messages to subscribers, and unsubscribe from events.

### Features:

- Lightweight, in-memory event handling.
- Supports multiple subscribers per event.
- Designed for use within a single application instance (not suitable for distributed systems).

### Example:

```go
package main

import (
    "fmt"
    "github.com/akiratatsuhisa/pubsub"
)

func main() {
    ps := pubsub.NewPubSub()

    ch := make(chan interface{})
    unsubscribe := ps.Subscribe("event1", ch)

    go func() {
        for msg := range ch {
            fmt.Println("Received:", msg)
        }
    }()

    ps.Publish("event1", "Hello, Subscriber!")

    // Unsubscribe when done
    unsubscribe()
}
```

## MongoDB Adapter PubSub

The MongoDB adapter provides a more durable solution for PubSub, where events are stored in a MongoDB collection. Subscribers can receive events published to the database, and the system supports automatic cleanup of expired events.

### Features:

Stores events in MongoDB with automatic expiration.
Allows subscriptions and listens for changes in the MongoDB collection.
Scalable and resilient for distributed systems.

### Example:

```go
package main

import (
    "context"
    "fmt"
    "github.com/akiratatsuhisa/pubsub"
)

func main() {
    opts := &pubsub.MongoPubSubOpts{
        Ctx:        context.Background(),
        Uri:        "mongodb://localhost:27017",
        DbName:     "pubsub_db",
        CollName:   "events",
        TtlSeconds: 7200 // 2 hours
    }

    ps := pubsub.NewMongoPubSub(opts)

    ch := make(chan []byte)
    unsubscribe := ps.Subscribe("event1", ch)

    go func() {
        for bytes := range ch {
            fmt.Println("Received:", string(bytes))
        }
    }()

    // Publish an event to MongoDB
    ps.Publish("event1", "Hello, MongoDB Subscriber!")

    // Unsubscribe when done
    unsubscribe()
}

```

### MongoDB Indexing

The MongoDB PubSub implementation creates an index on the `timestamp` field of the `PubSubDocument` to ensure automatic expiration of events using a TTL (Time-To-Live). The TTL value is configurable through the `MongoPubSubOpts` options, allowing users to set their desired expiration time for events in seconds. By default, if no TTL is provided, the TTL is set to 3600 seconds (1 hour).

You can customize the TTL by passing the `TtlSeconds` option in the `MongoPubSubOpts` struct when initializing the `MongoPubSub` instance.

## Installation

To install the package, run:

```bash
go get github.com/akiratatsuhisa/pubsub
```

## Usage

Import the package into your project:

```go
import "github.com/akiratatsuhisa/pubsub"
```

## License

This package is released under the [MIT license](LICENSE).

## Contributing

Feel free to open issues and submit pull requests for improvements, bug fixes, or feature requests. Make sure to follow the existing code style and write tests for new features or bug fixes.
