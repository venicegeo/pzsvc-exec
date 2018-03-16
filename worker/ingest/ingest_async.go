package ingest

import (
	"errors"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// asyncIngestor is an interface providing mock-able ingestFileAsync functionality, for modularity/testing purposes
type asyncIngestor interface {
	ingestFileAsync(s pzsvc.Session, filePath, fileType, serviceID, algVersion string, attMap map[string]string) <-chan singleIngestOutput
}

type defaultAsyncIngestor struct{}

func (ingestor defaultAsyncIngestor) ingestFileAsync(s pzsvc.Session, filePath, fileType, serviceID, algVersion string, attMap map[string]string) <-chan singleIngestOutput {
	outChan := make(chan singleIngestOutput)

	// Nested goroutines to allow for 1 minute for ingestion to succeed
	go func() {
		resultChan := make(chan singleIngestOutput)
		go func() {
			dataID, err := pzSvcIngestorInstance.IngestFile(s, filePath, fileType, serviceID, algVersion, attMap)

			resultChan <- singleIngestOutput{
				FilePath: filePath,
				DataID:   dataID,
				Error:    err,
			}
			close(resultChan)
		}()
		select {
		case result := <-resultChan:
			outChan <- result
		case <-pzSvcIngestorInstance.Timeout():
			outChan <- singleIngestOutput{
				FilePath: filePath,
				Error:    errors.New("File ingest timed out"),
			}
		}
		close(outChan)
	}()

	return outChan
}

var asyncIngestorInstance asyncIngestor = &defaultAsyncIngestor{}
