# `PulsarClient.CreateConsumer` — design notes

Internal reference for [`internal/jobs/pulsar/client.go`](../../internal/jobs/pulsar/client.go). The executor uses a separate consumer type — [`PulsarJobConsumer`](../../internal/jobs/pulsar/consumer.go) — with explicit `Receive()` loops and ack/nack policy.

## What the method does

`CreateConsumer` subscribes to a topic with a **Shared** subscription (`PULSAR_SUBSCRIPTION_NAME`, default `jobber`), registers the consumer for shutdown tracking, and delivers messages to a caller-supplied `messageHandler` on a background goroutine.

Rough flow:

1. Allocate `ch := make(chan pulsar.ConsumerMessage)`.
2. `client.Subscribe` with `MessageChannel: ch` — the Pulsar client pushes each message onto `ch`.
3. Start a goroutine that reads `ch` and invokes `messageHandler(consumer, message)` per message.
4. Under `c.mu`, append the consumer to `c.consumers` (for `Close()`).
5. Return the `pulsar.Consumer` immediately (non-blocking API).

## Channel + goroutine pattern

```go
go func() {
    for cm := range ch {
        messageHandler(cm.Consumer, cm.Message)
    }
}()
```

| Piece | Role |
|-------|------|
| `MessageChannel` | Tells the official client to deliver via `ch` instead of only `Receive()`. |
| `go func() { ... }()` | `CreateConsumer` returns without blocking in a receive loop. |
| `for cm := range ch` | Blocks until the next message; exits when `ch` is closed (typically after `consumer.Close()`). |
| `messageHandler(...)` | Application logic (decode `JobMessage`, ack/nack). Runs **synchronously** in this goroutine. |

**Implications**

- **One goroutine per consumer** — messages for that subscription are handled sequentially unless the handler spawns its own work.
- **Backpressure** — a slow `messageHandler` keeps the goroutine busy; the channel and client `ReceiverQueueSize` can fill and slow broker delivery.
- **Panics** — an uncaught panic in `messageHandler` terminates only that goroutine; there is no recovery in the loop.
- **Not Java `MessageListener`** — the Go client has no equivalent callback API; channel + loop (or explicit `Receive()` as in `PulsarJobConsumer`) is the idiomatic approach.

## Mutex: manual `Unlock()` vs `defer`

The critical section is only the slice append:

```go
c.mu.Lock()
c.consumers = append(c.consumers, consumer)
c.mu.Unlock()
```

For this shape, **`Unlock()` and `defer c.mu.Unlock()` right after `Lock` are equivalent** — same lock duration, same correctness.

What matters is **where** the lock lives:

- **Current (correct):** lock **after** `Subscribe` and after starting the listener goroutine. Mutex is held only for the append (~nanoseconds).
- **Wrong:** `Lock` + `defer Unlock` at the **top** of `CreateConsumer` would hold `c.mu` across `Subscribe` (network I/O) and goroutine setup — longer contention and risk of deadlocks if anything under the lock called back into `PulsarClient`.

Use `defer` locally around the append if the guarded region grows (multiple returns). Do **not** widen the critical section to cover `Subscribe` or handler setup.

`CreateProducer` uses the same short lock-then-unlock pattern for the same reason.

## Shutdown and lifecycle

- `Close()` closes tracked consumers, then producers, then the client (see `client.go`).
- Closing a consumer should close `MessageChannel`, which ends the `for range` loop and stops the listener goroutine.
- `CreateConsumer` does not wire signal handling here; callers or higher-level wrappers (such as `PulsarJobConsumer.Run`) own context cancellation and ack/nack policy.

## Related configuration

- **Subscription:** `config.PulsarConfig.SubscriptionName` (`PULSAR_SUBSCRIPTION_NAME`, default `jobber`).
- **Type:** `pulsar.Shared` — round-robin across consumers with the same subscription name; at-least-once delivery.
- **Producer backpressure (separate):** `CreateProducer` sets `DisableBlockIfQueueFull: false` so `Send` blocks when the pending queue is full rather than failing fast or dropping.

## See also

- [`PulsarJobConsumer`](../../internal/jobs/pulsar/consumer.go) — executor receive loop, ack/nack, and multi-topic subscription.
- [Pulsar Go client docs](https://pulsar.apache.org/docs/client-libraries-go-use/) — `Subscribe`, `MessageChannel`, Shared subscriptions.
