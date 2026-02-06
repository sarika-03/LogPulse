package ratelimiter

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"

	"github.com/logpulse/backend/internal/config"
)

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}

	return limiter
}

func Middleware(cfg *config.RateLimitConfig) mux.MiddlewareFunc {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	requestsPerSecond := float64(cfg.RequestsPerMinute) / 60.0
	limiter := NewIPRateLimiter(rate.Limit(requestsPerSecond), cfg.Burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractIP(r)

			if isWhitelisted(ip, cfg.WhitelistIPs) {
				log.Printf("[Rate Limit] Bypassed for whitelisted IP: %s", ip)
				next.ServeHTTP(w, r)
				return
			}

			if isBlacklisted(ip, cfg.BlacklistIPs) {
				log.Printf("[Rate Limit] Access denied for blacklisted IP: %s", ip)
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			lim := limiter.GetLimiter(ip)
			if !lim.Allow() {
				log.Printf("[Rate Limit] Exceeded for IP: %s (limit: %d req/min, burst: %d)",
					ip, cfg.RequestsPerMinute, cfg.Burst)

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

func extractIP(r *http.Request) string {
	ip := r.RemoteAddr

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
		ip = strings.TrimSpace(ip)
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		ip = realIP
	}
	if lastColon := strings.LastIndex(ip, ":"); lastColon != -1 {
		ip = ip[:lastColon]
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
