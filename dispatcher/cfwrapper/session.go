package cfwrapper

import (
	"net/url"

	cfclient "github.com/venicegeo/go-cfclient"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// CFSession is an abstraction around the Go CF client library to make its inclusion modular
type CFSession interface {
	IsValid() (bool, error)
	CountTasksForApp(appID string) (int, error)
	CreateTask(request TaskRequest) error
}

// CFWrappedSession is an implementation of CFSession that wraps an actual cfclient.Client
type wrappedCFSession struct {
	PzSession *pzsvc.Session
	Client    *cfclient.Client
}

// IsValid checks if the session's client is live and useful
func (s wrappedCFSession) IsValid() (bool, error) {
	if s.Client == nil {
		return false, nil
	}

	//Configure a query that will filter out everything.
	validationQueryParams := url.Values{}
	validationQueryParams.Add("q", "name:dummy_name")

	pzsvc.LogInfo(*s.PzSession, "Submitting client validation request.")
	_, err := s.Client.ListAppsByQuery(validationQueryParams)
	pzsvc.LogInfo(*s.PzSession, "Validation request complete.")
	if err == nil {
		return true, nil
	}

	if cfclient.IsNotAuthorizedError(err) || cfclient.IsNotAuthenticatedError(err) {
		return false, nil
	}
	return false, err
}

// CountTasksForApp returns the number of currently running tasks for a CF app
func (s wrappedCFSession) CountTasksForApp(appID string) (int, error) {
	query := url.Values{}
	query.Add("states", "RUNNING")
	tasks, err := s.Client.TasksByAppByQuery(appID, query)
	if err != nil {
		pzsvc.LogSimpleErr(*s.PzSession, "Cannot poll CF tasks", err)
		return 0, err
	}
	return len(tasks), nil
}

// CreateTask is a simple wrapper around cfclient.Client.CreateTask
func (s wrappedCFSession) CreateTask(request TaskRequest) error {
	_, err := s.Client.CreateTask(cfclient.TaskRequest(request))
	return err
}

func newWrappedCFSession(pzSession *pzsvc.Session, config *FactoryConfig) (CFSession, error) {
	client, err := cfclient.NewClient(config.CFClientConfig())
	if err != nil {
		return nil, err
	}
	return &wrappedCFSession{
		PzSession: pzSession,
		Client:    client,
	}, nil
}
