package ingest

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setUpMockPzSvcIngestor()
	setUpMockAsyncIngestor()
	setUpMockOutputs()

	retCode := m.Run()

	tearDownMockOutputs()
	tearDownMockAsyncIngestor()
	tearDownMockPzSvcIngestor()
	os.Exit(retCode)
}
