package crossid

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pomerium/pomerium/pkg/grpc/directory"
)

// A ServiceAccount is used by the Crossid provider to query Crossid API management.
type ServiceAccount struct {
	ClientID string `json:"client_id"`
	Secret   string `json:"secret"`
}

// ParseServiceAccount parses the service account in the config options.
func ParseServiceAccount(options directory.Options) (*ServiceAccount, error) {
	if options.ServiceAccount != "" {
		return decodeServiceAccountFromBase64(options.ServiceAccount)
	}
	return parseServiceAccountFromOptions(options.ClientID, options.ClientSecret)
}

// parseServiceAccountFromOptions creates a ServiceAccount instance by clientID and clientSecret
func parseServiceAccountFromOptions(clientID, clientSecret string) (*ServiceAccount, error) {
	if clientID == "" {
		return nil, ErrClientIdRequired
	}
	if clientSecret == "" {
		return nil, ErrClientSecretRequired
	}

	return &ServiceAccount{
		ClientID: clientID,
		Secret:   clientSecret,
	}, nil
}

// decodeServiceAccountFromBase64 decodes svcAccountB64 as a ServiceAccount instance
func decodeServiceAccountFromBase64(svcAccountB64 string) (*ServiceAccount, error) {
	b, err := base64.StdEncoding.DecodeString(svcAccountB64)
	if err != nil {
		return nil, fmt.Errorf("%s: could not decode base64: %w", Name, err)
	}

	var serviceAccount ServiceAccount
	if err := json.Unmarshal(b, &serviceAccount); err != nil {
		return nil, fmt.Errorf("%s: could not unmarshal json: %w", Name, err)
	}

	if serviceAccount.ClientID == "" {
		return nil, ErrClientIdRequired
	}

	if serviceAccount.Secret == "" {
		return nil, ErrClientSecretRequired
	}

	return &serviceAccount, nil
}
