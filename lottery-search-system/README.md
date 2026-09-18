# Lottery Search System Design

Design proposal for searching and allocating lottery tickets with wildcard patterns.

This section is design-only; no code implementation is required.

## Overview

The system contains 10M+ physical lottery tickets. Each ticket has a 6-digit number.

There are only 1,000,000 possible 6-digit numbers, so the number itself is not a unique ticket identity. Each physical ticket has its own `ticket_id`, and multiple tickets may share the same number.

Search results display distinct lottery numbers, not physical ticket IDs. A number remains searchable while at least one physical ticket for that number is available.

## Storage Choice

Use PostgreSQL as the primary database because it provides:

- B-tree and bitmap index support
- Transactions and strong consistency
- Row-level locking
- `FOR UPDATE SKIP LOCKED`
- Operational simplicity

Partitioning, sharding, Redis locks, and a separate search engine are not required initially.

## Data Model

```text
lottery_numbers
---------------
id
number
d1, d2, d3, d4, d5, d6
available_count
random_bucket

tickets
-------
id
number_id
status              available | reserved | purchased
reserved_by
reserved_until
```

`available_count` is denormalized so wildcard searches can query the at-most 1M distinct numbers without scanning the full 10M+ physical-ticket inventory.

`random_bucket` is a stable hash-derived value used to distribute results without expensive `ORDER BY RANDOM()` queries.

## Wildcard Search

A pattern such as `12**56` is parsed into fixed-position predicates:

```text
d1 = 1
d2 = 2
d5 = 5
d6 = 6
```

The query filters only fixed positions and available numbers:

```sql
SELECT id, number
FROM lottery_numbers
WHERE available_count > 0
  AND d1 = 1
  AND d2 = 2
  AND d5 = 5
  AND d6 = 6;
```

PostgreSQL's planner chooses the execution plan based on indexes and statistics; predicate order does not force index usage.

## Indexing Strategy

Prefix searches use a forward composite index:

```sql
CREATE INDEX idx_lottery_forward
ON lottery_numbers (d1, d2, d3, d4, d5, d6);
```

Useful for patterns such as `123***`, `12****`, and `1*****`.

Suffix searches use a reverse composite index:

```sql
CREATE INDEX idx_lottery_reverse
ON lottery_numbers (d6, d5, d4, d3, d2, d1);
```

Useful for patterns such as `***456`, `****56`, and `*****6`.

For arbitrary wildcard positions, position-specific indexes may be used:

```sql
CREATE INDEX idx_lottery_d1 ON lottery_numbers (d1);
CREATE INDEX idx_lottery_d2 ON lottery_numbers (d2);
CREATE INDEX idx_lottery_d3 ON lottery_numbers (d3);
CREATE INDEX idx_lottery_d4 ON lottery_numbers (d4);
CREATE INDEX idx_lottery_d5 ON lottery_numbers (d5);
CREATE INDEX idx_lottery_d6 ON lottery_numbers (d6);
```

For a pattern such as `*2*4*6`, PostgreSQL may combine multiple indexes using bitmap scans.

Do not create a composite index for every wildcard combination. Retain position-specific indexes only when production measurements show that they improve the actual workload.

For physical ticket allocation:

```sql
CREATE INDEX idx_tickets_available_by_number
ON tickets (number_id, id)
WHERE status = 'available';
```

## Result Distribution and Pagination

Search returns distinct lottery numbers. Each search session collects up to 300 matching and available candidate numbers.

Instead of running `ORDER BY RANDOM()` across a large result set, the service uses `random_bucket` to sample candidates from different parts of the matching population. A deterministic seed defines the bucket traversal order.

```text
matching population
        |
random bucket traversal
        |
up to 300 candidates
        |
service-level deterministic shuffle
        |
50 results per page
```

Pagination uses a page size of 50 and a maximum of 6 pages. The same search session and seed are reused across pages so the result order remains stable and results do not repeat within the session.

The 300-result batch is a bounded pseudo-random sample, not a materialized copy of every matching row.

## Concurrent Reservation

Search does not reserve physical tickets. Multiple users may see the same lottery number while multiple physical tickets remain available.

When a user selects a number, the service reserves one physical ticket atomically:

```sql
BEGIN;

SELECT id
FROM tickets
WHERE number_id = $1
  AND status = 'available'
FOR UPDATE SKIP LOCKED
LIMIT 1;

UPDATE tickets
SET status = 'reserved',
    reserved_by = $2,
    reserved_until = NOW() + INTERVAL '5 minutes'
WHERE id = $ticket_id;

UPDATE lottery_numbers
SET available_count = available_count - 1
WHERE id = $number_id;

COMMIT;
```

If two users select the same number concurrently, row-level locking gives them different physical tickets:

```text
123456

T001 available
T002 available

User A -> T001
User B -> T002
```

If only one ticket remains, one request succeeds and the other receives an unavailable result. The reservation transaction is the source of truth; search results may become stale between presentation and selection.

## Reservation Expiration

Expired reservations return to the available pool:

```text
reserved -> available
```

The release operation updates both the physical ticket and `available_count` in the same transaction. A background worker can reclaim expired reservations in batches.

Payment and order workflows are outside the scope of this design.

## Performance

Wildcard searches operate primarily on `lottery_numbers`, which contains at most 1,000,000 distinct numbers, rather than scanning the full physical-ticket table.

The `tickets` table is accessed only when a user selects a number.

For approximately uniform digit distribution:

| Pattern | Approximate matching numbers |
| --- | ---: |
| `1*****` | 10% |
| `12****` | 1% |
| `123***` | 0.1% |
| `1234**` | 0.01% |

Validate actual query plans with:

```sql
EXPLAIN (ANALYZE, BUFFERS)
```

Do not assume every index will always be selected.

## Scaling and Trade-offs

- Partitioning is not introduced initially because wildcard patterns do not provide one consistently useful partition key.
- Partitioning by the first digit helps `1*****` but provides little benefit for `****23` or `*2*4*6`.
- Partitioning can be introduced later using a natural dimension such as `draw_id`, `campaign_id`, or `draw_date`.
- `available_count` improves search performance but introduces denormalized state, so ticket state changes and count updates must occur in the same PostgreSQL transaction.
- Additional indexes improve reads but increase storage and write-maintenance cost. Indexes should be driven by real query patterns and production measurements.

## Scope

Included:

- Wildcard search over 6-digit lottery numbers
- Distinct-number result distribution
- Stable pagination within a search session
- Concurrent physical-ticket reservation
- Reservation expiration

Not included:

- Payment processing
- Order workflows
- gRPC implementation
- Code implementation
