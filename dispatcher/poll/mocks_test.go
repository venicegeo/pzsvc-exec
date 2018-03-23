package poll

import (
	"os"

	"github.com/venicegeo/pzsvc-exec/dispatcher/cfwrapper"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

type originalEnv struct {
	key           string
	originalValue string
}

func setMockEnv(key, value string) *originalEnv {
	env := &originalEnv{key, os.Getenv(key)}
	os.Setenv(key, value)
	return env
}

func (e originalEnv) Restore() {
	os.Setenv(e.key, e.originalValue)
}

type originalPzsvcGetS3FileSizeInMegabytesFunc func(string) (int, *pzsvc.PzCustomError)

func setMockPzsvcGetS3FileSizeInMegabytes(mockFunc func(string) (int, *pzsvc.PzCustomError)) originalPzsvcGetS3FileSizeInMegabytesFunc {
	original := pzsvcGetS3FileSizeInMegabytes
	pzsvcGetS3FileSizeInMegabytes = mockFunc
	return original
}

func (f originalPzsvcGetS3FileSizeInMegabytesFunc) Restore() {
	pzsvcGetS3FileSizeInMegabytes = f
}

type originalPzsvcRequestKnownJSONFunc func(string, string, string, string, interface{}) ([]byte, *pzsvc.PzCustomError)

func setMockPzsvcRequestKnownJSON(mockFunc func(string, string, string, string, interface{}) ([]byte, *pzsvc.PzCustomError)) originalPzsvcRequestKnownJSONFunc {
	original := pzsvcRequestKnownJSON
	pzsvcRequestKnownJSON = mockFunc
	return original
}

func (f originalPzsvcRequestKnownJSONFunc) Restore() {
	pzsvcRequestKnownJSON = f
}

type originalPzsvcSendExecResultNoDataFunc func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError

func setMockPzsvcSendExecResultNoData(mockFunc func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError) originalPzsvcSendExecResultNoDataFunc {
	original := pzsvcSendExecResultNoData
	pzsvcSendExecResultNoData = mockFunc
	return original
}

func (f originalPzsvcSendExecResultNoDataFunc) Restore() {
	pzsvcSendExecResultNoData = f
}

type mockCFWrapperFactory struct {
	Session                   cfwrapper.CFSession
	GetSessionError           error
	RefreshCachedSessionError error
}

func (m mockCFWrapperFactory) GetSession() (cfwrapper.CFSession, error) {
	return m.Session, m.GetSessionError
}

func (m mockCFWrapperFactory) RefreshCachedClient() error {
	return m.RefreshCachedSessionError
}

type mockCFSession struct {
	Valid           bool
	NumTasks        int
	IsValidError    error
	CountTasksError error
	CreateTaskError error
}

func (m mockCFSession) IsValid() (bool, error) {
	return m.Valid, m.IsValidError
}

func (m mockCFSession) CountTasksForApp(appID string) (int, error) {
	return m.NumTasks, m.CountTasksError
}

func (m mockCFSession) CreateTask(request cfwrapper.TaskRequest) error {
	return m.CreateTaskError
}
