package cfwrapper

import (
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// Factory encapsulates functionality to lazily generate CFSession objects
type Factory struct {
	pzSession     *pzsvc.Session
	config        *FactoryConfig
	cachedSession CFSession
	createSession createSessionFunc
}

// NewFactory creates a new factory object for lazily creating cfclient.Client objects
func NewFactory(pzSession *pzsvc.Session, config *FactoryConfig) *Factory {
	return &Factory{
		pzSession:     pzSession,
		config:        config,
		createSession: config.createSessionFunc,
	}
}

// GetSession returns a lazily generated CFSession, verified for validity
func (f *Factory) GetSession() (CFSession, error) {
	refreshSession := false

	if f.cachedSession == nil {
		refreshSession = true
	} else if valid, err := f.cachedSession.IsValid(); err != nil {
		return nil, err
	} else if !valid {
		refreshSession = true
	}

	if refreshSession {
		err := f.RefreshCachedClient()
		if err != nil {
			return nil, err
		}
	}

	return f.cachedSession, nil
}

// RefreshCachedClient replaces the cached client with a new one based on the
// factory's stored cfclient.Config and expiration duration
func (f *Factory) RefreshCachedClient() error {
	pzsvc.LogInfo(*f.pzSession, "Regenerating Cloud Foundry Client.")

	session, err := f.createSession(f.pzSession, f.config)
	if err == nil {
		f.cachedSession = session
	}
	return err
}