package cfwrapper

import (
	"net/http"

	cfclient "github.com/venicegeo/go-cfclient"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"golang.org/x/oauth2"
)

type createSessionFunc func(*pzsvc.Session, *FactoryConfig) (CFSession, error)

// TaskRequest is a rename of cfclient.TaskRequest to reduce imports
type TaskRequest cfclient.TaskRequest

// FactoryConfig is an expansion of cfclient.FactoryConfig to reduce imports and inject a session creation function
type FactoryConfig struct {
	APIAddress        string
	Username          string
	Password          string
	HTTPClient        *http.Client
	Token             string
	TokenSource       oauth2.TokenSource
	createSessionFunc createSessionFunc
}

// CFClientConfig converts this factory config to a cfclient-compatible configuration
func (fc FactoryConfig) CFClientConfig() *cfclient.Config {
	return &cfclient.Config{
		ApiAddress:  fc.APIAddress,
		Username:    fc.Username,
		Password:    fc.Password,
		HttpClient:  fc.HTTPClient,
		Token:       fc.Token,
		TokenSource: fc.TokenSource,
		UserAgent:   "venicegeo/pzsvc-exec/dispatcher/cfwrapper 2.0",
	}
}
