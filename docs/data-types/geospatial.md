---
title: "Geospatial"
---

Geospatial types use latitude/longitude coordinates and spatial predicates.

See [Type construction and operations](operations.md#geospatial-operations) for coordinate validation, Haversine distance, 0.25-degree grid candidate selection, exact post-filtering, result ordering, and polygon/antimeridian limitations.

| Type                                              | Construction | Principal operations              |
| ------------------------------------------------- | ------------ | --------------------------------- |
| [`geo_point`](types/geo_point.md)                 | SET          | DISTANCE                          |
| [`geo_spatial_index`](types/geo_spatial_index.md) | CREATE       | ADD, REMOVE, RADIUS, BOX, POLYGON |
| [`bounding_box`](types/bounding_box.md)           | SET          | CONTAINS                          |
| [`polygon`](types/polygon.md)                     | SET          | CONTAINS                          |
