package ingest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/pzsvc-worker/config"
)

const ingestTimeout = 1 * time.Minute

// OutputFilesToPiazza ingests the given files into the Piazza system
func OutputFilesToPiazza(cfg config.WorkerConfig, algFullCommand string, algVersion string) error {
	errorTexts := []string{}
	ingestErrChans := []<-chan error{}

	for _, filePath := range cfg.Outputs {
		if _, fStatErr := os.Stat(filePath); fStatErr != nil {
			errorTexts = append(errorTexts, fmt.Sprintf("file does not exist (%s)", filePath))
			continue
		}

		fileType := detectPiazzaFileType(filePath)

		attMap := map[string]string{
			"algoName":     cfg.PiazzaServiceID,
			"algoVersion":  algVersion,
			"algoCmd":      algFullCommand,
			"algoProcTime": time.Now().UTC().Format("20060102.150405.99999"),
		}

		errChan := ingestFileAsync(*cfg.Session, filePath, fileType, cfg.PiazzaServiceID, algVersion, attMap)
		ingestErrChans = append(ingestErrChans, errChan)
	}

	for _, errChan := range ingestErrChans {
		for err := range errChan {
			if err != nil {
				errorTexts = append(errorTexts, err.Error())
			}
		}
	}

	if len(errorTexts) > 0 {
		fullErrorText := strings.Join(errorTexts, "; ")
		return errors.New("ingest errors: " + fullErrorText)
	}

	return nil
}

func ingestFileAsync(s pzsvc.Session, filePath string, fileType string,
	serviceID string, algVersion string, attMap map[string]string) <-chan error {
	outChan := make(chan error)

	// Nested goroutines to allow for 1 minute for ingestion to succeed
	go func() {
		resultChan := make(chan error)
		go func() {
			_, err := pzsvc.IngestFile(s, filePath, fileType, serviceID, algVersion, attMap)
			resultChan <- err
			close(resultChan)
		}()
		select {
		case err := <-resultChan:
			outChan <- err
		case <-time.After(ingestTimeout):
			outChan <- errors.New("File ingest timed out")
		}
		close(outChan)
	}()

	return outChan
}

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
