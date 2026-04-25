# Distributed Redis Rate Limiter

A small Go proof-of-concept for a distributed **token bucket rate limiter** backed by Redis. The target is to enforce per-identity request budgets (for example `userID`, IP, or API token) across multiple app instances without relying on in-memory state, background refill jobs, or non-atomic coordination.

## What it does

- Uses a **Redis Lua script** to make refill + consume decisions atomically.
- Stores per-key limiter state as `tokens` and `last_refill` in a Redis hash.
- Applies **lazy refill** on request arrival using `elapsed * refill_rate`, capped by `max_tokens`.
- Returns a simple allow/deny decision suitable for `HTTP 429` integration.

## Production snapshot

- Productionized Redis-backed token bucket limiter with atomic Lua execution.
- Enforces per-identity budgets and returns `200` / `429` for API integration.
- Works across multiple app instances with shared Redis state.

## Load test metrics

Command:
`hey -n 1000 -c 50 http://localhost:8080/api/data`

Results (latest run):
- **Requests/sec:** `13100.63`
- **Average latency:** `3.4ms` (p95 `8.7ms`, p99 `16.7ms`)
- **Status codes:** `200 = 5`, `429 = 995`
- **Total runtime:** `76.3ms`

Expected behavior: the initial burst consumed the configured bucket capacity (`5` tokens), and the remaining concurrent requests were correctly throttled with `429`.

This confirms strict rate-limit enforcement under concurrent burst traffic.