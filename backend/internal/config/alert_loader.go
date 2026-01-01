package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

type AlertRule struct {
	Name      string            `yaml:"name" json:"name"`
	Expr      string            `yaml:"expr" json:"expr"`
	Threshold float64           `yaml:"threshold" json:"threshold"`
	Window    string            `yaml:"window" json:"window"`
	Channels  []string          `yaml:"channels" json:"channels"`
	Labels    map[string]string `yaml:"labels" json:"labels"`
}

type AlertSettings struct {
	Alerts []AlertRule `yaml:"alerts" json:"alerts"`
}

func LoadAlerts(path string) ([]AlertRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var as AlertSettings
	if err := yaml.Unmarshal(data, &as); err != nil {
		return nil, err
	}
	return as.Alerts, nil
}
