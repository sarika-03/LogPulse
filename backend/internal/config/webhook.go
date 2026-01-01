package config

type WebhookConfig struct {
	URL    string   `yaml:"url" json:"url"`
	Events []string `yaml:"events" json:"events"`
}

type WebhookSettings struct {
	Webhooks []WebhookConfig `yaml:"webhooks" json:"webhooks"`
}
