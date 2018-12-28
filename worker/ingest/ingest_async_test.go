package ingest

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// test setup/teardown

type mockPzSvcIngestorCall struct {
	s                                 pzsvc.Session
	fName, fType, sourceName, version string
	props                             map[string]string
}

type mockPzSvcIngestor struct {
	Calls        []mockPzSvcIngestorCall
	CauseTimeout bool
	ReturnFileID string
	ReturnError  pzsvc.LoggedError
}

func (ingestor *mockPzSvcIngestor) IngestFile(s pzsvc.Session, fName, fType, sourceName, version string, props map[string]string) (string, pzsvc.LoggedError) {
	ingestor.Calls = append(ingestor.Calls, mockPzSvcIngestorCall{s, fName, fType, sourceName, version, props})
	if ingestor.CauseTimeout {
		blockedChan := make(chan time.Time)
		<-blockedChan
	}
	return ingestor.ReturnFileID, ingestor.ReturnError
}

func (ingestor mockPzSvcIngestor) Timeout() <-chan time.Time {
	// if we are causing a timeout, create a channel that returns a time immediately
	// otherwise, never return a time
	returnChan := make(chan time.Time, 1)
	if ingestor.CauseTimeout {
		returnChan <- time.Now()
	}
	return returnChan
}

func (ingestor *mockPzSvcIngestor) Reset(causeTimeout bool, returnFileID string, returnError pzsvc.LoggedError) {
	ingestor.Calls = []mockPzSvcIngestorCall{}
	ingestor.CauseTimeout = causeTimeout
	ingestor.ReturnFileID = returnFileID
	ingestor.ReturnError = returnError
}

var mockPzSvcIngestorInstance *mockPzSvcIngestor

func setUpMockPzSvcIngestor() {
	mockPzSvcIngestorInstance = &mockPzSvcIngestor{}
	mockPzSvcIngestorInstance.Reset(false, "", nil)
	pzSvcIngestorInstance = mockPzSvcIngestorInstance
}

func tearDownMockPzSvcIngestor() {
	pzSvcIngestorInstance = &defaultPzSvcIngestor{}
}

// actual test functions

func TestIngestFileAsync_Success(t *testing.T) {
	// Setup
	mockPzSvcIngestorInstance.Reset(false, "testReturnFileID", nil)

	// Tested code
	ingestResult := <-defaultAsyncIngestor{}.ingestFileAsync(pzsvc.Session{}, "path/to/output/file", "geojson", "service-id-123", "alg-version-0.1", map[string]string{})

	// Asserts
	assert.Equal(t, "path/to/output/file", mockPzSvcIngestorInstance.Calls[0].fName)
	assert.Equal(t, "path/to/output/file", ingestResult.FilePath)
	assert.Equal(t, "testReturnFileID", ingestResult.DataID)
	assert.Nil(t, ingestResult.Error)
}

func TestIngestFileAsync_Failure(t *testing.T) {
	// Setup
	var loggedError pzsvc.LoggedError = errors.New("test error")
	mockPzSvcIngestorInstance.Reset(false, "", loggedError)

	// Tested code
	ingestResult := <-defaultAsyncIngestor{}.ingestFileAsync(pzsvc.Session{}, "path/to/output/file", "geojson", "service-id-123", "alg-version-0.1", map[string]string{})

	// Asserts
	assert.Equal(t, "path/to/output/file", mockPzSvcIngestorInstance.Calls[0].fName)
	assert.Equal(t, "path/to/output/file", ingestResult.FilePath)
	assert.Equal(t, "", ingestResult.DataID)
	assert.Equal(t, loggedError, ingestResult.Error)
}

func TestIngestFileAsync_Timeout(t *testing.T) {
	// Setup
	mockPzSvcIngestorInstance.Reset(true, "testReturnFileID", nil)

	// Tested code
	ingestResult := <-defaultAsyncIngestor{}.ingestFileAsync(pzsvc.Session{}, "path/to/output/file", "geojson", "service-id-123", "alg-version-0.1", map[string]string{})

	// Asserts
	assert.Equal(t, "path/to/output/file", ingestResult.FilePath)
	assert.Equal(t, "", ingestResult.DataID)
	assert.Contains(t, ingestResult.Error.Error(), "Unexpected error storing job output")
}
