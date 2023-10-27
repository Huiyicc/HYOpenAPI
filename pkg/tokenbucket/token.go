package tokenbucket

import (
	"sync"
	"time"
)

type TokenBucket struct {
	rate       float64
	capacity   float64
	tokens     float64
	lastFilled time.Time
	mutex      sync.Mutex
}

// NewTokenBucket 创建一个新的流量桶
//
//	rate: 每秒生成x个令牌
//	capacity: 令牌桶的容量为x个令牌
func NewTokenBucket(rate float64, capacity float64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastFilled: time.Now(),
	}
}

// fillTokens 根据距离上次填充令牌的时间以及令牌生成速率来填充令牌桶
func (tb *TokenBucket) fillTokens() {
	now := time.Now()
	delta := now.Sub(tb.lastFilled).Seconds()
	tb.tokens = tb.tokens + tb.rate*delta
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastFilled = now
}

// TryConsume 尝试从令牌桶中消耗一个令牌，返回成功或失败。
func (tb *TokenBucket) TryConsume() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.fillTokens()

	if tb.tokens >= 1 {
		tb.tokens = tb.tokens - 1
		return true
	}

	return false
}
