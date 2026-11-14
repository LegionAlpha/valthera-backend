package httpapi

import (
	"net/netip"
	"sync"
	"time"
)

const (
	registrationAttemptLimit = 5
	registrationWindow       = time.Hour
	registrationMaxIPs       = 1024
)

type registrationLimiter struct {
	mu       sync.Mutex
	attempts map[netip.Addr][]time.Time
}

func newRegistrationLimiter() *registrationLimiter {
	return &registrationLimiter{
		attempts: make(map[netip.Addr][]time.Time),
	}
}

func (l *registrationLimiter) allow(ip netip.Addr) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-registrationWindow)

	for key, attempts := range l.attempts {
		first := 0
		for first < len(attempts) &&
			!attempts[first].After(cutoff) {
			first++
		}

		if first == len(attempts) {
			delete(l.attempts, key)
		} else if first > 0 {
			l.attempts[key] = attempts[first:]
		}
	}

	attempts, exists := l.attempts[ip]

	if !exists && len(l.attempts) >= registrationMaxIPs {
		return false
	}

	if len(attempts) >= registrationAttemptLimit {
		return false
	}

	l.attempts[ip] = append(attempts, now)
	return true
}
