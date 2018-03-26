package cfwrapper

import (
	"fmt"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

type Factory interface {
	GetSession() (CFSession, error)
	RefreshCachedClient() error
}

// DefaultFactory encapsulates functionality to lazily generate CFSession objects
type DefaultFactory struct {
	pzSession     *pzsvc.Session
	config        *FactoryConfig
	cachedSession CFSession
	createSession createSessionFunc
}

// NewFactory creates a new factory object for lazily creating cfclient.Client objects
func NewFactory(pzSession *pzsvc.Session, config *FactoryConfig) *DefaultFactory {
	return &DefaultFactory{
		pzSession:     pzSession,
		config:        config,
		createSession: config.createSessionFunc,
	}
}

// GetSession returns a lazily generated CFSession, verified for validity
func (f *DefaultFactory) GetSession() (CFSession, error) {
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
func (f *DefaultFactory) RefreshCachedClient() error {
	pzsvc.LogInfo(*f.pzSession, fmt.Sprintf("Regenerating Cloud Foundry Client with create function: %v", f.createSession))

	session, err := f.createSession(f.pzSession, f.config)
	if err == nil {
		f.cachedSession = session
	}
	return err
}
