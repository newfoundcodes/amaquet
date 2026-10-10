---
title: "Indexes and search"
---

Index/search types provide exact, range, prefix, full-text, vector, and secondary lookup paths.

See [Type construction and operations](operations.md#index-and-search-operations) for request contracts and implementation notes covering B-tree splits/rebuilds, byte-wise radix traversal, tokenization/ranking, and explicit index maintenance. Index objects are not updated automatically when another key changes.

| Type                                                                | Construction | Principal operations                          |
| ------------------------------------------------------------------- | ------------ | --------------------------------------------- |
| [`btree_index`](types/btree_index.md)                               | CREATE       | INSERT, DELETE, SEARCH, RANGE                 |
| [`hash_index`](types/hash_index.md)                                 | CREATE       | ADD, REMOVE, LOOKUP                           |
| [`radix_tree`](types/radix_tree.md)                                 | CREATE       | INSERT, EXACT, PREFIX                         |
| [`trie`](types/trie.md)                                             | CREATE       | INSERT, EXACT, PREFIX                         |
| [`inverted_index`](types/inverted_index.md)                         | CREATE       | INDEX, REMOVE, SEARCH                         |
| [`hnsw_index`](types/hnsw_index.md)                                 | CREATE       | ADD, REMOVE, SEARCH                           |
| [`flat_vector_index`](types/flat_vector_index.md)                   | CREATE       | ADD, REMOVE, SEARCH, LEN                      |
| [`geospatial_secondary_index`](types/geospatial_secondary_index.md) | CREATE       | ADD, REMOVE, RADIUS, BOX, POLYGON             |
| [`secondary_index`](types/secondary_index.md)                       | CREATE       | ADD, LOOKUP                                   |
| [`full_text_document`](types/full_text_document.md)                 | CREATE       | ADD_TEXT, ADD_TAG, ADD_NUMERIC, GET, SNAPSHOT |
| [`tag_field`](types/tag_field.md)                                   | CREATE       | SET, GET                                      |
| [`numeric_field`](types/numeric_field.md)                           | CREATE       | SET, GET                                      |
| [`text_field`](types/text_field.md)                                 | CREATE       | SET, GET                                      |
