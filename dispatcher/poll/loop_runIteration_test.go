package poll

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/dispatcher/cfwrapper"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// These tests are extensive for a complex function, so they are broken out into their own file

func TestRunIteration_ErrGetSession(t *testing.T) {
	// Setup
	loop := Loop{
		vcapID:        "test-vcap-id",
		SvcID:         "test-svc-id",
		PzSession:     &pzsvc.Session{},
		ClientFactory: &mockCFWrapperFactory{Session: mockCFSession{}, GetSessionError: errors.New("get session error")},
		taskLimit:     10,
	}

	originalGetS3FileSize := setMockPzsvcGetS3FileSizeInMegabytes(func(string) (int, *pzsvc.PzCustomError) { return 0, nil })
	defer originalGetS3FileSize.Restore()
	originalRequestJSON := setMockPzsvcRequestKnownJSON(func(string, string, string, string, interface{}) ([]byte, *pzsvc.PzCustomError) { return nil, nil })
	defer originalRequestJSON.Restore()
	originalSendExecResult := setMockPzsvcSendExecResultNoData(func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError { return nil })
	defer originalSendExecResult.Restore()

	// Test code
	err := runIteration(loop)

	// Asserts
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "get session error")
}

func TestRunIteration_ErrCountTasks(t *testing.T) {
	// Setup
	loop := Loop{
		vcapID:        "test-vcap-id",
		SvcID:         "test-svc-id",
		PzSession:     &pzsvc.Session{},
		ClientFactory: &mockCFWrapperFactory{Session: mockCFSession{CountTasksError: errors.New("count error")}},
		taskLimit:     10,
	}

	originalGetS3FileSize := setMockPzsvcGetS3FileSizeInMegabytes(func(string) (int, *pzsvc.PzCustomError) { return 0, nil })
	defer originalGetS3FileSize.Restore()
	originalRequestJSON := setMockPzsvcRequestKnownJSON(func(string, string, string, string, interface{}) ([]byte, *pzsvc.PzCustomError) { return nil, nil })
	defer originalRequestJSON.Restore()
	originalSendExecResult := setMockPzsvcSendExecResultNoData(func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError { return nil })
	defer originalSendExecResult.Restore()

	// Test code
	err := runIteration(loop)

	// Asserts
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "count error")
}

func TestRunIteration_TooManyTasks(t *testing.T) {
	// Setup
	loop := Loop{
		vcapID:        "test-vcap-id",
		SvcID:         "test-svc-id",
		PzSession:     &pzsvc.Session{},
		ClientFactory: &mockCFWrapperFactory{Session: mockCFSession{NumTasks: 11}},
		taskLimit:     10,
	}

	externalsCalled := 0
	originalGetS3FileSize := setMockPzsvcGetS3FileSizeInMegabytes(func(string) (int, *pzsvc.PzCustomError) {
		externalsCalled++
		return 0, nil
	})
	defer originalGetS3FileSize.Restore()
	originalRequestJSON := setMockPzsvcRequestKnownJSON(func(string, string, string, string, interface{}) ([]byte, *pzsvc.PzCustomError) {
		externalsCalled++
		return nil, nil
	})
	defer originalRequestJSON.Restore()
	originalSendExecResult := setMockPzsvcSendExecResultNoData(func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError {
		externalsCalled++
		return nil
	})
	defer originalSendExecResult.Restore()

	// Test code
	err := runIteration(loop)

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, 0, externalsCalled) // No external functions should be called to figure out we can't run more tasks
}

func TestRunIteration_ErrGetTask(t *testing.T) {
	// Setup
	loop := Loop{
		vcapID:        "test-vcap-id",
		SvcID:         "test-svc-id",
		PzSession:     &pzsvc.Session{},
		ClientFactory: &mockCFWrapperFactory{Session: mockCFSession{}},
		taskLimit:     10,
	}

	originalGetS3FileSize := setMockPzsvcGetS3FileSizeInMegabytes(func(string) (int, *pzsvc.PzCustomError) { return 0, nil })
	defer originalGetS3FileSize.Restore()
	originalRequestJSON := setMockPzsvcRequestKnownJSON(func(_, _, _, _ string, outObj interface{}) ([]byte, *pzsvc.PzCustomError) {
		return nil, &pzsvc.PzCustomError{LogMsg: "test piazza task error"}
	})
	defer originalRequestJSON.Restore()
	originalSendExecResult := setMockPzsvcSendExecResultNoData(func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError { return nil })
	defer originalSendExecResult.Restore()

	// Test code
	err := runIteration(loop)

	// Asserts
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test piazza task error")
}

func TestRunIteration_EmptyTaskContent(t *testing.T) {
	// Setup
	loop := Loop{
		vcapID:        "test-vcap-id",
		SvcID:         "test-svc-id",
		PzSession:     &pzsvc.Session{},
		ClientFactory: &mockCFWrapperFactory{Session: mockCFSession{}},
		taskLimit:     10,
	}

	var (
		s3FileSizeRequests = 0
		taskRequests       = 0
		sendResultRequests = 0
	)

	originalGetS3FileSize := setMockPzsvcGetS3FileSizeInMegabytes(func(string) (int, *pzsvc.PzCustomError) {
		s3FileSizeRequests++
		return 0, nil
	})
	defer originalGetS3FileSize.Restore()
	originalRequestJSON := setMockPzsvcRequestKnownJSON(func(_, _, _, _ string, outObj interface{}) ([]byte, *pzsvc.PzCustomError) {
		taskRequests++
		body := []byte(`{"data": {"serviceData": {"jobID": "test-job-id", "data": {"dataInputs": {"body": {"content": ""}}}}}}`)
		json.Unmarshal(body, outObj)
		return body, nil
	})
	defer originalRequestJSON.Restore()
	originalSendExecResult := setMockPzsvcSendExecResultNoData(func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError {
		sendResultRequests++
		return nil
	})
	defer originalSendExecResult.Restore()

	// Test code
	err := runIteration(loop)

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, 0, s3FileSizeRequests)
	assert.Equal(t, 1, taskRequests)
	assert.Equal(t, 0, sendResultRequests)
}

func TestRunIteration_BadTaskContent(t *testing.T) {
	// Setup
	loop := Loop{
		vcapID:        "test-vcap-id",
		SvcID:         "test-svc-id",
		PzSession:     &pzsvc.Session{},
		ClientFactory: &mockCFWrapperFactory{Session: mockCFSession{}},
		taskLimit:     10,
	}

	originalGetS3FileSize := setMockPzsvcGetS3FileSizeInMegabytes(func(string) (int, *pzsvc.PzCustomError) { return 0, nil })
	defer originalGetS3FileSize.Restore()
	originalRequestJSON := setMockPzsvcRequestKnownJSON(func(_, _, _, _ string, outObj interface{}) ([]byte, *pzsvc.PzCustomError) {
		body := []byte(`{"data": {"serviceData": {"jobID": "test-job-id", "data": {"dataInputs": {"body": {"content": "#"}}}}}}`)
		json.Unmarshal(body, outObj)
		return body, nil
	})
	defer originalRequestJSON.Restore()
	originalSendExecResult := setMockPzsvcSendExecResultNoData(func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError { return nil })
	defer originalSendExecResult.Restore()

	// Test code
	err := runIteration(loop)

	// Asserts
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid character '#'")
}

func TestRunIteration_BadJobInput(t *testing.T) {
	// Setup
	loop := Loop{
		vcapID:        "test-vcap-id",
		SvcID:         "test-svc-id",
		PzSession:     &pzsvc.Session{},
		ClientFactory: &mockCFWrapperFactory{Session: mockCFSession{}},
		taskLimit:     10,
	}

	originalGetS3FileSize := setMockPzsvcGetS3FileSizeInMegabytes(func(string) (int, *pzsvc.PzCustomError) { return 0, nil })
	defer originalGetS3FileSize.Restore()
	originalRequestJSON := setMockPzsvcRequestKnownJSON(func(_, _, _, _ string, outObj interface{}) ([]byte, *pzsvc.PzCustomError) {
		body := []byte(`{"data": {"serviceData": {"jobID": "test-job-id", "data": {"dataInputs": {"body": {"content": "{\"inExtFiles\": [\"http:\/\/input.localdomain\/foo.txt\"], \"inExtNames\": [\"outA.geojson\", \"outB.geojson\"]}"}}}}}}`)
		json.Unmarshal(body, outObj)
		return body, nil
	})
	defer originalRequestJSON.Restore()
	originalSendExecResult := setMockPzsvcSendExecResultNoData(func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError { return nil })
	defer originalSendExecResult.Restore()

	// Test code
	err := runIteration(loop)

	// Asserts
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "did not match") // XXX: this is kind of a bad check for a specific error already covered by another test
}

func TestRunIteration_ErrMemoryLimit(t *testing.T) {
	// Setup
	loop := Loop{
		vcapID:        "test-vcap-id",
		SvcID:         "test-svc-id",
		PzSession:     &pzsvc.Session{},
		ClientFactory: &mockCFWrapperFactory{Session: mockCFSession{CreateTaskError: &cfwrapper.CustomMemoryLimitError{Message: "test memory limit error"}}},
		taskLimit:     10,
	}

	originalGetS3FileSize := setMockPzsvcGetS3FileSizeInMegabytes(func(string) (int, *pzsvc.PzCustomError) { return 0, nil })
	defer originalGetS3FileSize.Restore()
	originalRequestJSON := setMockPzsvcRequestKnownJSON(func(_, _, _, _ string, outObj interface{}) ([]byte, *pzsvc.PzCustomError) {
		body := []byte(`{"data": {"serviceData": {"jobID": "test-job-id", "data": {"dataInputs": {"body": {"content": "{\"inExtFiles\": [\"http:\/\/input.localdomain\/foo.txt\"], \"inExtNames\": [\"output.geojson\"]}"}}}}}}`)
		json.Unmarshal(body, outObj)
		return body, nil
	})
	defer originalRequestJSON.Restore()
	originalSendExecResult := setMockPzsvcSendExecResultNoData(func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError { return nil })
	defer originalSendExecResult.Restore()

	// Test code
	err := runIteration(loop)

	// Asserts
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "CF memory limit hit")
}

func TestRunIteration_ErrUnknown(t *testing.T) {
	// Setup
	loop := Loop{
		vcapID:        "test-vcap-id",
		SvcID:         "test-svc-id",
		PzSession:     &pzsvc.Session{},
		ClientFactory: &mockCFWrapperFactory{Session: mockCFSession{CreateTaskError: errors.New("test unknown error")}},
		taskLimit:     10,
	}

	originalGetS3FileSize := setMockPzsvcGetS3FileSizeInMegabytes(func(string) (int, *pzsvc.PzCustomError) { return 0, nil })
	defer originalGetS3FileSize.Restore()
	originalRequestJSON := setMockPzsvcRequestKnownJSON(func(_, _, _, _ string, outObj interface{}) ([]byte, *pzsvc.PzCustomError) {
		body := []byte(`{"data": {"serviceData": {"jobID": "test-job-id", "data": {"dataInputs": {"body": {"content": "{\"inExtFiles\": [\"http:\/\/input.localdomain\/foo.txt\"], \"inExtNames\": [\"output.geojson\"]}"}}}}}}`)
		json.Unmarshal(body, outObj)
		return body, nil
	})
	defer originalRequestJSON.Restore()
	originalSendExecResult := setMockPzsvcSendExecResultNoData(func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus) *pzsvc.PzCustomError { return nil })
	defer originalSendExecResult.Restore()

	// Test code
	err := runIteration(loop)

	// Asserts
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "test unknown error")
}
