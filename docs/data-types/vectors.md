---
title: "Vectors"
---

Vector types support embeddings, metadata, quantization, and similarity search.

See [Type construction and operations](operations.md#vector-operations) for metric definitions, dimensions, exact-scan behavior, HNSW graph construction/search, quantization, result ordering, and the current protocol's lack of metadata-filter expressions.

| Type                                            | Construction | Principal operations     |
| ----------------------------------------------- | ------------ | ------------------------ |
| [`dense_vector`](types/dense_vector.md)         | SET          | DISTANCE                 |
| [`sparse_vector`](types/sparse_vector.md)       | SET          | GET                      |
| [`vector_set`](types/vector_set.md)             | CREATE       | ADD, REMOVE, SEARCH, LEN |
| [`vector_metadata`](types/vector_metadata.md)   | CREATE       | GET, SET, DELETE         |
| [`quantized_vector`](types/quantized_vector.md) | SET          | DEQUANTIZE               |
