---
title: "Bit structures"
---

Bit structures provide compact bit-level and integer-set storage.

See [Type construction and operations](operations.md#bit-structure-operations) for addressing, width/endian rules, return values, and the adaptive array/bitmap/run containers used by `roaring_bitmap`.

| Type                                        | Construction | Principal operations                       |
| ------------------------------------------- | ------------ | ------------------------------------------ |
| [`bitmap`](types/bitmap.md)                 | CREATE       | SET, GETBIT, COUNT                         |
| [`bit_field`](types/bit_field.md)           | CREATE       | SET, GET                                   |
| [`bit_set`](types/bit_set.md)               | CREATE       | SET, GETBIT, COUNT                         |
| [`roaring_bitmap`](types/roaring_bitmap.md) | CREATE       | ADD, REMOVE, CONTAINS, VALUES, CARDINALITY |
