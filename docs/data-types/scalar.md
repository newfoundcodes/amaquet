---
title: "Scalar"
---

Scalar values are stored directly in a key and use `SET`.

See [Type construction and operations](operations.md#scalar-and-direct-value-operations) for exact JSON encodings, arithmetic arguments/results, geospatial predicates, vector metrics, overflow/validation behavior, and which values have no type-specific `OP`.

| Type                                            | Construction | Principal operations |
| ----------------------------------------------- | ------------ | -------------------- |
| [`binary_string`](types/binary_string.md)       | SET          | GET                  |
| [`utf8_string`](types/utf8_string.md)           | SET          | GET                  |
| [`integer`](types/integer.md)                   | SET          | ADD                  |
| [`unsigned_integer`](types/unsigned_integer.md) | SET          | ADD                  |
| [`float`](types/float.md)                       | SET          | ADD                  |
| [`decimal`](types/decimal.md)                   | SET          | ADD, SUB, CMP        |
| [`boolean`](types/boolean.md)                   | SET          | TOGGLE               |
| [`null`](types/null.md)                         | SET          | GET                  |
| [`timestamp`](types/timestamp.md)               | SET          | GET                  |
| [`duration`](types/duration.md)                 | SET          | ADD                  |
| [`uuid`](types/uuid.md)                         | SET          | GET                  |
| [`big_integer`](types/big_integer.md)           | SET          | ADD                  |
| [`big_decimal`](types/big_decimal.md)           | SET          | ADD, SUB, CMP        |
| [`symbol`](types/symbol.md)                     | SET          | GET                  |
