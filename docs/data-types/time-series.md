---
title: "Time series"
---

Time-series types store timestamped numeric data and derived statistics.

See [Type construction and operations](operations.md#time-series-operations) for timestamp units, overwrite and retention behavior, bucket anchoring, supported aggregation modes, histogram buckets, and derived-series source handling.

| Type                                              | Construction | Principal operations             |
| ------------------------------------------------- | ------------ | -------------------------------- |
| [`timeseries`](types/timeseries.md)               | CREATE       | ADD, RANGE, LAST, AGGREGATE, LEN |
| [`counter_series`](types/counter_series.md)       | CREATE       | ADD, RANGE, LAST, AGGREGATE, LEN |
| [`gauge_series`](types/gauge_series.md)           | CREATE       | SET, RANGE, LAST, AGGREGATE, LEN |
| [`histogram`](types/histogram.md)                 | CREATE       | OBSERVE, SNAPSHOT                |
| [`timeseries_labels`](types/timeseries_labels.md) | CREATE       | GET, SET                         |
| [`aggregated_series`](types/aggregated_series.md) | CREATE       | RANGE, AGGREGATE                 |
