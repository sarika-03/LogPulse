package plugin

import (
	"sync"
	"time"
)

type AlertRule struct {
	Name      string            `json:"name"`
	Expr      string            `json:"expr"` // e.g. `{service="api"} |= "error" | count_over_time([5m]) > 10`
	Threshold float64           `json:"threshold"`
	Window    time.Duration     `json:"window"`
	Channels  []string          `json:"channels"` // e.g. ["slack", "webhook"]
	Labels    map[string]string `json:"labels"`
}

type AlertManager struct {
	Rules    []AlertRule
	mu       sync.RWMutex
	Notifier *WebhookNotifier
}

func NewAlertManager(notifier *WebhookNotifier) *AlertManager {
	return &AlertManager{
		Rules:    []AlertRule{},
		Notifier: notifier,
	}
}

func (am *AlertManager) AddRule(rule AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.Rules = append(am.Rules, rule)
}

// EvaluateRules should be called periodically (e.g. every minute)
func (am *AlertManager) EvaluateRules(queryFunc func(expr string) (float64, error)) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	for _, rule := range am.Rules {
		value, err := queryFunc(rule.Expr)
		if err == nil && value > rule.Threshold {
			if am.Notifier != nil {
				am.Notifier.Notify("alert", map[string]interface{}{
					"rule":      rule.Name,
					"expr":      rule.Expr,
					"value":     value,
					"labels":    rule.Labels,
					"channels":  rule.Channels,
					"timestamp": time.Now().Format(time.RFC3339),
				})
			}
		}
	}
}
