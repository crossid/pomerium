// Package crossid contains the Crossid directory provider.
package crossid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pomerium/pomerium/internal/log"
	"github.com/pomerium/pomerium/pkg/grpc/directory"
	"github.com/rs/zerolog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Name is the provider name.
const Name = "crossid"

const (
	// apiPrefix is crossid's api management API prefix
	apiPrefix = "/api/v1"
)

// Errors.
var (
	// ErrServiceAccountNotDefined is an error returned when no service account defined.
	ErrServiceAccountNotDefined = errors.New(fmt.Sprintf("%s: service account is not defined", Name))
	// ErrProviderURLNotDefined is an error returned when no provider URL defined.
	ErrProviderURLNotDefined = errors.New(fmt.Sprintf("%s: provider URL is not defined", Name))
	// ErrClientIdRequired is an error returned when no client_id defined.
	ErrClientIdRequired = errors.New(fmt.Sprintf("%s: client_id is required", Name))
	// ErrClientSecretREquired is an error returned when no client_secret defined.
	ErrClientSecretRequired = errors.New(fmt.Sprintf("%s: client_secret is required", Name))
)

type config struct {
	providerURL    *url.URL
	httpClient     *http.Client
	serviceAccount *ServiceAccount
	client         *http.Client
}

// Provider is an Auth0 user group directory provider.
type Provider struct {
	cfg *config
	log zerolog.Logger
}

// New creates a new Provider.
func New(options ...Option) *Provider {
	return &Provider{
		cfg: getConfig(options...),
		log: log.With().Str("service", "directory").Str("provider", Name).Logger(),
	}
}

func withLog(ctx context.Context) context.Context {
	return log.WithContext(ctx, func(c zerolog.Context) zerolog.Context {
		return c.Str("service", "directory").Str("provider", Name)
	})
}

// User returns the user record for the given id.
func (p *Provider) User(ctx context.Context, userID, accessToken string) (*directory.User, error) {
	ctx = withLog(ctx)

	if p.cfg.serviceAccount == nil {
		return nil, ErrServiceAccountNotDefined
	}

	du := &directory.User{
		Id: userID,
	}

	return nil, nil
}

// UserGroups fetches a slice of groups and users.
func (p *Provider) UserGroups(ctx context.Context) ([]*directory.Group, []*directory.User, error) {
	// TODO
	return nil, nil, nil
}

func (p *Provider) getUser(ctx context.Context, userID string) (*cidUser, error) {
	apiURL := p.cfg.providerURL.ResolveReference(&url.URL{
		Path: fmt.Sprintf("/users/%s", userID),
	}).String()

	var out cidUser
	_, err := p.apiGet(ctx, apiURL, &out)
	if err != nil {
		return nil, fmt.Errorf("okta: error querying for user: %w", err)
	}

	return &out, nil
}

func (p *Provider) apiGet(ctx context.Context, uri string, out interface{}) (http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create HTTP request: %w", Name, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", p.cfg.serviceAccount.Secret)

	for {
		res, err := p.cfg.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusTooManyRequests {
			limitReset, err := strconv.ParseInt(res.Header.Get("X-Rate-Limit-Reset"), 10, 64)
			if err == nil {
				time.Sleep(time.Until(time.Unix(limitReset, 0)))
			}
			continue
		}
		//if res.StatusCode/100 != httpSuccessClass {
		//	return nil, newAPIError(res)
		//}
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			return nil, err
		}
		return res.Header, nil
	}
}
