package ratelimiter

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"

	"github.com/logpulse/backend/internal/config"
)

type ipLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type IPRateLimiter struct {
	ips            map[string]*ipLimiterEntry
	mu             *sync.RWMutex
	r              rate.Limit
	b              int
	cleanupTicker  *time.Ticker
	ttl            time.Duration
	done           chan struct{}
	trustedProxies map[string]bool
}

func NewIPRateLimiter(r rate.Limit, b int, trustedProxies []string) *IPRateLimiter {
	limiter := &IPRateLimiter{
		ips:            make(map[string]*ipLimiterEntry),
		mu:             &sync.RWMutex{},
		r:              r,
		b:              b,
		cleanupTicker:  time.NewTicker(5 * time.Minute),
		ttl:            10 * time.Minute,
		done:           make(chan struct{}),
		trustedProxies: make(map[string]bool),
	}

	for _, proxy := range trustedProxies {
		limiter.trustedProxies[proxy] = true
	}

	go limiter.cleanupLoop()

	return limiter
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	entry, exists := i.ips[ip]
	if !exists {
		entry = &ipLimiterEntry{
			limiter:    rate.NewLimiter(i.r, i.b),
			lastAccess: time.Now(),
		}
		i.ips[ip] = entry
	} else {
		entry.lastAccess = time.Now()
	}

	return entry.limiter
}

func (i *IPRateLimiter) cleanupLoop() {
	for {
		select {
		case <-i.cleanupTicker.C:
			i.cleanup()
		case <-i.done:
			i.cleanupTicker.Stop()
			return
		}
	}
}

func (i *IPRateLimiter) cleanup() {
	i.mu.Lock()
	defer i.mu.Unlock()

	now := time.Now()
	for ip, entry := range i.ips {
		if now.Sub(entry.lastAccess) > i.ttl {
			delete(i.ips, ip)
		}
	}
}

func (i *IPRateLimiter) Stop() {
	close(i.done)
}

func Middleware(cfg *config.RateLimitConfig) mux.MiddlewareFunc {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	requestsPerSecond := float64(cfg.RequestsPerMinute) / 60.0
	limiter := NewIPRateLimiter(rate.Limit(requestsPerSecond), cfg.Burst, cfg.TrustedProxies)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractIP(r, limiter.trustedProxies)

			if isWhitelisted(ip, cfg.WhitelistIPs) {
				log.Printf("[Rate Limit] Bypassed for whitelisted IP: %s", maskIP(ip))
				next.ServeHTTP(w, r)
				return
			}

			if isBlacklisted(ip, cfg.BlacklistIPs) {
				log.Printf("[Rate Limit] Access denied for blacklisted IP: %s", maskIP(ip))
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			lim := limiter.GetLimiter(ip)
			if !lim.Allow() {
				log.Printf("[Rate Limit] Exceeded for IP: %s (limit: %d req/min, burst: %d)",
					maskIP(ip), cfg.RequestsPerMinute, cfg.Burst)

				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.RequestsPerMinute))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
				w.Header().Set("Retry-After", "60")

				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractIP(r *http.Request, trustedProxies map[string]bool) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	if len(trustedProxies) > 0 && trustedProxies[ip] {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			if len(parts) > 0 {
				candidateIP := strings.TrimSpace(parts[0])
				if net.ParseIP(candidateIP) != nil {
					return candidateIP
				}
			}
		}

		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			if net.ParseIP(realIP) != nil {
				return realIP
			}
		}
	}

	return ip
}

func maskIP(ip string) string {
	if parsedIP := net.ParseIP(ip); parsedIP != nil && parsedIP.To4() != nil {
		parts := strings.Split(ip, ".")
		if len(parts) == 4 {
			return fmt.Sprintf("%s.%s.%s.xxx", parts[0], parts[1], parts[2])
		}
	}

	if parsedIP := net.ParseIP(ip); parsedIP != nil && parsedIP.To4() == nil {
		parts := strings.Split(ip, ":")
		if len(parts) > 2 {
			return parts[0] + ":" + parts[1] + ":xxxx"
		}
	}

	return ip
}

func isWhitelisted(ip string, whitelist []string) bool {
	for _, whitelistIP := range whitelist {
		if ip == whitelistIP {
			return true
		}
	}
	return false
}

func isBlacklisted(ip string, blacklist []string) bool {
	for _, blacklistIP := range blacklist {
		if ip == blacklistIP {
			return true
		}
	}
	return false
}
