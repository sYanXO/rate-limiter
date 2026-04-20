package main

import (
	"fmt"
	"sync"
	"time"
)

// define the bucketState

type Config struct { // separating concerns cuz these are global
	MaxToken   float64
	Refillrate float64
}
type TokenBucket struct {
	Tokens         float64
	LastRefillTime time.Time
	Mu             sync.Mutex
	Config         Config
}

func NewTokenBucket(config Config) *TokenBucket { // constructor
	return &TokenBucket{
		Tokens:         config.MaxToken,
		LastRefillTime: time.Now(),
		Config:         config,
	}
}

// refill func take in param as a tokenbucket
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.LastRefillTime).Seconds()

	refillTokens := elapsed * tb.Config.Refillrate

	tb.Tokens += refillTokens

	if tb.Tokens > tb.Config.MaxToken {
		tb.Tokens = tb.Config.MaxToken
	}

	tb.LastRefillTime = now
}

// allow() func takes in how many req came in returns if they are allowed to pass or not
func (tb *TokenBucket) allow(n float64) bool {
	tb.Mu.Lock()
	defer tb.Mu.Unlock()

	tb.refill()

	if tb.Tokens >= n {
		tb.Tokens -= n
		return true
	}

	return false
}

func main() {
	fmt.Println("token bucket rate limiter demo")

	tb := NewTokenBucket((Config{
		MaxToken:   5, // play around
		Refillrate: 1, // play around
	}))

	for i := 0; i < 25; i++ {
		if tb.allow(1) {
			fmt.Printf("Request %d : allowed\n", i+1)
		} else {
			fmt.Printf("Request %d : not allowed\n", i+1)
		}

		time.Sleep(200 * time.Millisecond) // request rate
	}
}

// add redis
// functioning backend
// stress test hard
// maybe atomic
