# Token Bucket Rate Limiting

## The Core Idea

Token bucket is not about users waiting. It is about rejecting requests that exceed the rate limit.

Think of it like this:

- You have a bucket with a maximum capacity, for example `10` tokens.
- Tokens refill at a constant rate, for example `1` token per second.
- Each request consumes `1` token.
- If the bucket is empty, the request is rejected with `HTTP 429`.
- If the bucket has tokens, the request is allowed and one token is consumed.

## The Trick: Lazy Refilling

You do not refill tokens every second with a background job. That is inefficient and does not scale.

Instead, you calculate tokens on demand when a request comes in:

```txt
tokens_to_add = (current_time - last_refill_time) * refill_rate
new_token_count = min(current_tokens + tokens_to_add, max_capacity)
```

## Concrete Example

### Config

- Max capacity: `10` tokens
- Refill rate: `2` tokens/second

### Timeline

#### T = 0s

Bucket starts full with `10` tokens.

#### T = 1s

One request comes in.

- Time since last refill: `1s`
- Tokens to add: `1s * 2 tokens/s = 2 tokens`
- Current: `10` tokens, already at max
- Allow request, consume `1` token
- Tokens left: `9`
- Update `last_refill_time = 1s`

#### T = 2s

Five requests come in rapid-fire.

- Time since last refill: `1s`
- Tokens to add: `1s * 2 = 2 tokens`
- Current: `9 + 2 = 11`, but cap at `10`
- Allow all `5`, consume `5`
- Tokens left: `5`
- Update `last_refill_time = 2s`

#### T = 2.5s

Six requests come in.

- Time since last refill: `0.5s`
- Tokens to add: `0.5s * 2 = 1 token`
- Current: `5 + 1 = 6 tokens`
- Allow first `6`, consume `6`
- Tokens left: `0`
- Update `last_refill_time = 2.5s`

#### T = 2.6s

One more request comes in.

- Time since last refill: `0.1s`
- Tokens to add: `0.1s * 2 = 0.2 tokens`
- Current: `0 + 0.2 = 0.2 tokens`
- Reject the request because at least `1` full token is required
- Do not update the timestamp because the request failed

#### T = 5s

Another request comes in.

- Time since last refill: `2.5s`, measured from `T = 2.5s`, the last successful refill
- Tokens to add: `2.5s * 2 = 5 tokens`
- Current: `0 + 5 = 5 tokens`
- Allow the request, consume `1`
- Tokens left: `4`

## What You Store Per User in Redis

```go
type RateLimitState struct {
    Tokens         float64   // Current token count; can be fractional.
    LastRefillTime time.Time // When refill was last calculated.
}
```

## The Algorithm in Pseudocode

```go
func AllowRequest(userID string, maxTokens, refillRate float64) bool {
    // Get current state from Redis.
    state := redis.Get(userID)

    // Calculate tokens to add since last refill.
    now := time.Now()
    elapsed := now.Sub(state.LastRefillTime).Seconds()
    tokensToAdd := elapsed * refillRate

    // Refill bucket, but do not exceed max capacity.
    state.Tokens = min(state.Tokens+tokensToAdd, maxTokens)
    state.LastRefillTime = now

    // Check whether the request can be allowed.
    if state.Tokens >= 1.0 {
        state.Tokens -= 1.0
        redis.Set(userID, state)
        return true
    }

    // Not enough tokens.
    redis.Set(userID, state)
    return false
}
```

## Why This Is Beautiful

- No background jobs. Refill happens lazily when requests come in.
- Handles bursts. If a user is idle, the bucket fills to max and they can burst later.
- Smooth long-term rate. Over time, the average rate is enforced by `refill_rate`.
- Fractional tokens. Timing stays precise without coarse rounding.

## Common Gotcha

**Q:** What if the user does not make requests for `10` minutes?

**A:** When they finally make a request, you calculate the tokens to add as `10 min * refill_rate`, but you cap the result at `max_capacity`. They do not accumulate infinite tokens.

## Redis Structure

- Key: `"ratelimit:user123"`
- Value: `{"tokens": 7.3, "last_refill": 1713707891.234}`
