// Rate limit for tool calls

package ratelimit

import (
	"sync"
	"time"
)

type TokenBucket struct {
	capacity   int
	tokens     int
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

// New token bucket
func NewTockenBucket(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a token is available
func (tb *TokenBucket) Allow() (bool, time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += int(elapsed * tb.refillRate)
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now

	if tb.tokens > 0 {
		tb.tokens--
		return true, 0
	}

	return false, time.Duration(float64(time.Second) / tb.refillRate)
}

// Sliding window rate limiter
type SlidingWindowLimiter struct {
	limit      int
	window     time.Duration
	timestamps []time.Time
	mu         sync.Mutex
}

// New sliding window limiter
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{limit: limit, window: window}
}

// Allow checks if request is within limits
func (sw *SlidingWindowLimiter) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)
	var valid []time.Time
	for _, t := range sw.timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	sw.timestamps = valid

	if len(sw.timestamps) >= sw.limit {
		return false
	}
	sw.timestamps = append(sw.timestamps, now)
	return true
}
