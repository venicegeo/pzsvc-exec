package ingest

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

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
				Error:    errors.New("Unexpected error storing job output"),
			}
		}
		close(outChan)
	}()

	return outChan
}

var asyncIngestorInstance asyncIngestor = &defaultAsyncIngestor{}

// pzSvcIngestor is an interface providing mock-able pzsvc.IngestFile functionality, for modularity/testing purposes
type pzSvcIngestor interface {
	IngestFile(s pzsvc.Session, fName, fType, sourceName, version string, props map[string]string) (string, pzsvc.LoggedError)
	Timeout() <-chan time.Time
}

type defaultPzSvcIngestor struct{}

func (ingestor defaultPzSvcIngestor) IngestFile(s pzsvc.Session, fName, fType, sourceName, version string, props map[string]string) (string, pzsvc.LoggedError) {
	return pzsvc.IngestFile(s, fName, fType, sourceName, version, props)
}

func (ingestor defaultPzSvcIngestor) Timeout() <-chan time.Time {
	return time.After(3 * time.Minute)
}

var pzSvcIngestorInstance pzSvcIngestor = &defaultPzSvcIngestor{}

func detectPiazzaFileType(fileName string) string {
	ext := filepath.Ext(strings.ToLower(fileName))

	switch ext {
	case ".geojson":
		return "geojson"
	case ".tiff", ".geotiff":
		return "raster"
	default:
		return "text"
	}
}
