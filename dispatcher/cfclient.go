package main

import (
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

// CFClientFactory encapsulates functionality to lazily generate cfclient.Client objects
type CFClientFactory struct {
	config                   *cfclient.Config
	cachedClient             *cfclient.Client
	cachedClientCreationTime time.Time
}

// NewCFClientFactory creates a new factory object for lazily creating cfclient.Client objects
func NewCFClientFactory(config *cfclient.Config) (*CFClientFactory, error) {
	clientFactory := &CFClientFactory{config: config}
	err := clientFactory.RefreshCachedClient()
	if err != nil {
		return nil, err
	}
	return clientFactory, nil
}

// GetClient returns a lazily generated *cfclient.Client, verified for validity
// using IsCachedClientExpired()
func (f *CFClientFactory) GetClient() (*cfclient.Client, error) {
	isExpired, err := f.IsCachedClientExpired()
	if err != nil {
		return nil, err
	}

	if isExpired {
		err = f.RefreshCachedClient()
		if err != nil {
			return nil, err
		}
	}

	return f.cachedClient, nil
}

// CachedClientAge returns the age of the cached client (the time.Duration since
// it was instantiated)
func (f *CFClientFactory) CachedClientAge() time.Duration {
	return time.Now().Sub(f.cachedClientCreationTime)
}

// RefreshCachedClient replaces the cached client with a new one based on the
// factory's stored cfclient.Config and expiration duration
func (f *CFClientFactory) RefreshCachedClient() error {
	client, err := cfclient.NewClient(f.config)
	if err != nil {
		return err
	}
	f.cachedClient = client
	f.cachedClientCreationTime = time.Now()
	return nil
}

// IsCachedClientExpired returns whether the cached client returns an authentication
// error when faced with a simple request; if it receives another error, it
// returns that instead
func (f CFClientFactory) IsCachedClientExpired() (bool, error) {
	if f.cachedClient == nil {
		return true, nil
	}
	_, err := f.cachedClient.ListApps()
	if err == nil {
		return false, nil
	}

	if cfclient.IsNotAuthorizedError(err) || cfclient.IsNotAuthenticatedError(err) {
		return true, nil
	}
	return false, err
}
