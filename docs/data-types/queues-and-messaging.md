---
title: "Queues and messaging"
---

Queue and messaging types provide ordering, delivery, streaming, and publication semantics.

See [Type construction and operations](operations.md#queue-operations) and [Stream and messaging operations](operations.md#stream-and-messaging-operations) for ordering, lease/dead-letter behavior, stream group state, retention, delivery guarantees, and exact request fields.

| Type                                            | Construction | Principal operations                                |
| ----------------------------------------------- | ------------ | --------------------------------------------------- |
| [`fifo_queue`](types/fifo_queue.md)             | CREATE       | ENQUEUE, DEQUEUE, PEEK, LEN                         |
| [`lifo_stack`](types/lifo_stack.md)             | CREATE       | PUSH, POP, PEEK, LEN                                |
| [`priority_queue`](types/priority_queue.md)     | CREATE       | ENQUEUE, DEQUEUE, LEN                               |
| [`blocking_queue`](types/blocking_queue.md)     | CREATE       | ENQUEUE, DEQUEUE, LEN, CLOSE                        |
| [`delayed_queue`](types/delayed_queue.md)       | CREATE       | ENQUEUE, POP_READY, LEN                             |
| [`reliable_queue`](types/reliable_queue.md)     | CREATE       | ENQUEUE, CLAIM, ACK, NACK, STATS                    |
| [`stream`](types/stream.md)                     | CREATE       | ADD, RANGE, GROUP_CREATE, GROUP_READ, ACK, PENDING… |
| [`consumer_group`](types/consumer_group.md)     | CREATE       | SNAPSHOT                                            |
| [`pubsub`](types/pubsub.md)                     | CREATE       | PUBLISH, STATS                                      |
| [`persistent_topic`](types/persistent_topic.md) | CREATE       | PUBLISH, READ                                       |
| [`event_log`](types/event_log.md)               | CREATE       | APPEND, READ, LEN                                   |
