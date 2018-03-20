package cfwrapper

import (
	"net/url"

	cfclient "github.com/venicegeo/go-cfclient"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// TaskRequest is a rename of cfclient.TaskRequest to reduce imports
type TaskRequest cfclient.TaskRequest

// CFSession is an abstraction around the Go CF client library to make its inclusion modular
type CFSession interface {
	IsValid() (bool, error)
	CountTasksForApp(appID string) (int, error)
	CreateTask(request TaskRequest) error
	IsMemoryLimitError(err error) bool
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

	_, err := s.Client.ListApps()
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

func (s wrappedCFSession) IsMemoryLimitError(err error) bool {
	return cfclient.IsAppMemoryQuotaExceededError(err) ||
		cfclient.IsQuotaInstanceLimitExceededError(err) ||
		cfclient.IsQuotaInstanceMemoryLimitExceededError(err) ||
		cfclient.IsSpaceQuotaInstanceMemoryLimitExceededError(err)
}

func newWrappedCFSession(pzSession *pzsvc.Session, config *cfclient.Config) (CFSession, error) {
	client, err := cfclient.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &wrappedCFSession{
		PzSession: pzSession,
		Client:    client,
	}, nil
}
