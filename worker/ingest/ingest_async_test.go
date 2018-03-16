package ingest

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// test setup/teardown

func TestMain(m *testing.M) {
	setUpMockIngestor()
	retCode := m.Run()
	// teardown here
	os.Exit(retCode)
}

type ingestorCall struct {
	s                                 pzsvc.Session
	fName, fType, sourceName, version string
	props                             map[string]string
}

type mockIngestor struct {
	Calls        []ingestorCall
	CauseTimeout bool
	ReturnFileID string
	ReturnError  pzsvc.LoggedError
}

func (ingestor *mockIngestor) IngestFile(s pzsvc.Session, fName, fType, sourceName, version string, props map[string]string) (string, pzsvc.LoggedError) {
	ingestor.Calls = append(ingestor.Calls, ingestorCall{s, fName, fType, sourceName, version, props})
	if ingestor.CauseTimeout {
		blockedChan := make(chan time.Time)
		<-blockedChan
	}
	return ingestor.ReturnFileID, ingestor.ReturnError
}

func (ingestor mockIngestor) Timeout() <-chan time.Time {
	// if we are causing a timeout, create a channel that returns a time immediately
	// otherwise, never return a time
	returnChan := make(chan time.Time, 1)
	if ingestor.CauseTimeout {
		returnChan <- time.Now()
	}
	return returnChan
}

func (ingestor *mockIngestor) Reset(causeTimeout bool, returnFileID string, returnError pzsvc.LoggedError) {
	ingestor.Calls = []ingestorCall{}
	ingestor.CauseTimeout = causeTimeout
	ingestor.ReturnFileID = returnFileID
	ingestor.ReturnError = returnError
}

var mockIngestorInstance *mockIngestor

func setUpMockIngestor() {
	mockIngestorInstance = &mockIngestor{}
	mockIngestorInstance.Reset(false, "", nil)
	pzSvcIngestorInstance = mockIngestorInstance
}

// actual test functions

func TestIngestFileAsync_Success(t *testing.T) {
	mockIngestorInstance.Reset(false, "testReturnFileID", nil)

	ingestResult := <-defaultAsyncIngestor{}.ingestFileAsync(pzsvc.Session{}, "path/to/output/file", "geojson", "service-id-123", "alg-version-0.1", map[string]string{})

	assert.Equal(t, "path/to/output/file", mockIngestorInstance.Calls[0].fName)
	assert.Equal(t, "path/to/output/file", ingestResult.FilePath)
	assert.Equal(t, "testReturnFileID", ingestResult.DataID)
	assert.Nil(t, ingestResult.Error)
}

func TestIngestFileAsync_Failure(t *testing.T) {
	var loggedError pzsvc.LoggedError = errors.New("test error")
	mockIngestorInstance.Reset(false, "", loggedError)

	ingestResult := <-defaultAsyncIngestor{}.ingestFileAsync(pzsvc.Session{}, "path/to/output/file", "geojson", "service-id-123", "alg-version-0.1", map[string]string{})

	assert.Equal(t, "path/to/output/file", mockIngestorInstance.Calls[0].fName)
	assert.Equal(t, "path/to/output/file", ingestResult.FilePath)
	assert.Equal(t, "", ingestResult.DataID)
	assert.Equal(t, loggedError, ingestResult.Error)
}

func TestIngestFileAsync_Timeout(t *testing.T) {
	mockIngestorInstance.Reset(true, "testReturnFileID", nil)

	ingestResult := <-defaultAsyncIngestor{}.ingestFileAsync(pzsvc.Session{}, "path/to/output/file", "geojson", "service-id-123", "alg-version-0.1", map[string]string{})

	assert.Equal(t, "path/to/output/file", ingestResult.FilePath)
	assert.Equal(t, "", ingestResult.DataID)
	assert.Contains(t, ingestResult.Error.Error(), "timed out")
}
