package ingest

import (
	"time"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// pzSvcIngestor is an interface providing a mock-able pzsvc.IngestFile functionality, for modularity/testing purposes
type pzSvcIngestor interface {
	IngestFile(s pzsvc.Session, fName, fType, sourceName, version string, props map[string]string) (string, pzsvc.LoggedError)
	Timeout() <-chan time.Time
}

type defaultIngestor struct{}

func (ingestor defaultIngestor) IngestFile(s pzsvc.Session, fName, fType, sourceName, version string, props map[string]string) (string, pzsvc.LoggedError) {
	return pzsvc.IngestFile(s, fName, fType, sourceName, version, props)
}

func (ingestor defaultIngestor) Timeout() <-chan time.Time {
	return time.After(1 * time.Minute)
}

var ingestor pzSvcIngestor = &defaultIngestor{}
