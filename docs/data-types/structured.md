---
title: "Structured"
---

Structured values cover documents, encoded payloads, records, matrices, and tensors.

See [Type construction and operations](operations.md#structured-operations) for schemas, shape rules, indexed access, and snapshot behavior. MessagePack and CBOR are opaque bytes; Amaquet does not validate or query those external encodings. JSON, MessagePack, and CBOR can be initialized empty with `CREATE` or populated directly with `SET`.

| Type                                  | Construction | Principal operations |
| ------------------------------------- | ------------ | -------------------- |
| [`json`](types/json.md)               | CREATE / SET | GET, SET             |
| [`messagepack`](types/messagepack.md) | CREATE / SET | GET                  |
| [`cbor`](types/cbor.md)               | CREATE / SET | GET                  |
| [`record`](types/record.md)           | CREATE       | SET, GET             |
| [`matrix`](types/matrix.md)           | CREATE       | GET, SET, LEN        |
| [`tensor`](types/tensor.md)           | CREATE       | GET, SET, LEN        |
