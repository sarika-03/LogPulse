package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

func LoadWebhooks(path string) ([]WebhookConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ws WebhookSettings
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return nil, err
	}
	return ws.Webhooks, nil
}
