package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const luaRateLimitScript = `
	local key = KEYS[1]
	local max_tokens = tonumber(ARGV[1])
	local refill_rate = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])
	local requested = tonumber(ARGV[4])


	local state = redis.call("HMGET",key,"tokens","last_refill")
	local tokens = tonumber(state[1])
	local last_refill = tonumber(state[2])


	if tokens == nil then
		tokens = max_tokens
		last_refill = now

	else
		local elapsed = math.max(0, now - last_refill)
		local refill = elapsed * refill_rate
		tokens = math.min(max_tokens, tokens + refill)
		last_refill = now
	end

	local allowed = 0
	if tokens >= requested then
		tokens = tokens - requested
		allowed = 1
	end

	redis.call("HMSET",key,"tokens",tokens,"last_refill",last_refill)
	redis.call("EXPIRE",key, math.ceil(max_tokens / refill_rate) + 60)

	return allowed
`

type RedisRateLimiter struct {
	client *redis.Client
	script *redis.Script
}

func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		script: redis.NewScript(luaRateLimitScript),
	}
}

func (rl *RedisRateLimiter) Allow(ctx context.Context, key string, maxTokens float64, refillRate float64, requested float64) (bool, error) {
	now := float64(time.Now().UnixNano()) / 1e9 // unix gives in seconds but time.now is in nanoseconds, its a hacky way to fix this indifference
	result, err := rl.script.Run(ctx, rl.client, []string{key}, maxTokens, refillRate, now, requested).Result()

	if err != nil {
		return false, err
	}
	return result.(int64) == 1, nil

}

// ratelimit middlewqrawe

func RateLimitMiddleware(limiter *RedisRateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			clientIP = r.RemoteAddr // Fallback just in case
		}
		key := "ratelimit:" + clientIP

		allowed, err := limiter.Allow(ctx, key, 5, 1, 1)

		if err != nil {
			fmt.Printf("Redis Error : %v", err)
			next(w, r)
			return
		}

		if !allowed {
			http.Error(w, "429 Too many Requests", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// secver code

func main() {

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	limiter := NewRedisRateLimiter(rdb)

	helloWorldHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // same as res.send("message") in TS
		w.Write([]byte("Success!"))
	}

	http.HandleFunc("/api/data", RateLimitMiddleware(limiter, helloWorldHandler))

	fmt.Println("Server runing on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}

}

// add redis - done
// functioning backend
// stress test hard
// maybe atomic
