---
title: "Adding a data type"
---

A new stable type touches several contracts. Keep them synchronized.

## 1. Register the type

Add the `DataType` constant and wire name in `internal/core/datatype.go`. Stable IDs must not be reordered after release.

## 2. Implement the value

Add the concrete implementation under `internal/types`. Define synchronization inside the value when operations can be concurrent.

## 3. Add construction

Use `decodeTypedValue` for direct `SET` construction or `createValue` for stateful `CREATE` construction.

## 4. Add operations

Extend `Dispatcher.applyOp` and update read-operation classification so RBAC/AOF behavior is correct.

## 5. Serialization

Update `publicData` when the concrete type needs a custom public representation.

## 6. Tests

Update the factory construction-path test and the Bash type manifest/contracts. Add Go unit tests for internal invariants and race-prone behavior.

## 7. Documentation

Add or regenerate the type reference and list creation options, wire shape, operations, complexity/invariants where relevant, and examples.
