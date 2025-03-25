# Watermill SQLite3 Driver Pack

Golang SQLite3 driver pack for Watermill event dispatcher. Drivers satisfy message Publisher and Subscriber interfaces.

## Development: Alpha Version

SQLite3 does not support querying `FOR UPDATE`, which is used for row locking when subscribers in the same consumer group read an event batch in official Watermill SQL PubSub implementations.

Current architectural decision is to lock a consumer group offset using `unixepoch()+graceTimeout` time stamp. While one consumed message is processing per group, the offset lock time is extended by `graceTimeout` periodically by `time.Ticker`. If the subscriber is unable to finish the consumer group batch, other subscribers will take over lock as soon as the grace period runs out.

- [ ] Finish time-based lock extension when:
    - [ ] sending a message to output channel
    - [ ] waiting for message acknowledgement
- [ ] Drop `subscription` in favor of handling output channels all within `subscriber` like the official SQL implementation? The tickers are needlessly duplicated - there is enough of one per consumer group. Likewise, only one writer can acquire a row lock, so duplicate SQL strings are sitting idle in parallel subscribers.
- [ ] Ephemeral in-memory `Connector` must also satisfy tests.
- [ ] add `NewDeduplicator` constructor for deduplication middleware.
- Pass official implementation acceptance tests:
    - [x] tests.TestPublishSubscribe
    - [x] tests.TestConcurrentSubscribe
    - [x] tests.TestConcurrentSubscribeMultipleTopics
    - [x] tests.TestResendOnError
    - [x] tests.TestNoAck
    - [x] tests.TestContinueAfterSubscribeClose
    - [x] tests.TestConcurrentClose
    - [x] tests.TestContinueAfterErrors
    - [x] tests.TestPublishSubscribeInOrder
    - [x] tests.TestPublisherClose
    - [x] tests.TestTopic
    - [x] tests.TestSubscribeCtx
    - [x] tests.TestNewSubscriberReceivesOldMessages
    - [ ] tests.TestMessageCtx
    - [ ] consumer group tests

## Similar Projects

- <https://github.com/walterwanderley/watermill-sqlite>
- <https://github.com/ov2b/watermill-sqlite3>
