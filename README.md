# Distributed Redis Rate Limiter

A small Go proof-of-concept for a distributed **token bucket rate limiter** backed by Redis. The target is to enforce per-identity request budgets (for example `userID`, IP, or API token) across multiple app instances without relying on in-memory state, background refill jobs, or non-atomic coordination.

## What it does

- Uses a **Redis Lua script** to make refill + consume decisions atomically.
- Stores per-key limiter state as `tokens` and `last_refill` in a Redis hash.
- Applies **lazy refill** on request arrival using `elapsed * refill_rate`, capped by `max_tokens`.
- Returns a simple allow/deny decision suitable for `HTTP 429` integration.

## Current status

- Core limiter implementation exists in `main.go`.
- Redis client wiring and script execution are in place.
- A demo loop exercises the limiter against a single key (`ratelimit:user1`) to show burst handling and rejections.
- The conceptual token bucket behavior is documented in `idea.md`.

## Notes

- The current prototype is intentionally minimal and focuses on the distributed control-plane logic.
- Time handling, script return typing, and production hardening are still areas to refine.
