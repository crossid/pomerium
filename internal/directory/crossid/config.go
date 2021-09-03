package crossid

import "net/url"

// Option provides config for the Crossid Provider.
type Option func(cfg *config)

// WithServiceAccount sets the service account option.
func WithServiceAccount(serviceAccount *ServiceAccount) Option {
	return func(cfg *config) {
		cfg.serviceAccount = serviceAccount
	}
}

// WithProviderURL sets the provider URL option.
func WithProviderURL(uri *url.URL) Option {
	return func(cfg *config) {
		cfg.providerURL = uri
	}
}

func getConfig(options ...Option) *config {
	cfg := &config{}
	for _, option := range options {
		option(cfg)
	}
	return cfg
}
